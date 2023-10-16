package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/air-gases/authenticator"
	"github.com/aofei/air"
	"github.com/google/uuid"

	"github.com/kdudkov/goatak/pkg/tlsutil"
)

const (
	usernameKey = "username"
	p12Password = "atakatak"
)

func getUsernameFromReq(req *air.Request) string {
	if u := req.Value(usernameKey); u != nil {
		return u.(string)
	}
	return ""
}

func getCertApi(app *App, addr string) *air.Air {
	certApi := air.New()
	certApi.Address = addr

	auth := authenticator.BasicAuthGas(authenticator.BasicAuthGasConfig{
		Validator: func(username string, password string, req *air.Request, _ *air.Response) (bool, error) {
			app.Logger.Infof("tls api login with user %s", username)
			req.SetValue(usernameKey, username)
			return app.users.CheckUserAuth(username, password), nil
		},
	})

	certApi.Gases = []air.Gas{auth}

	certApi.GET("/Marti/api/tls/config", getTlsConfigHandler(app))
	certApi.POST("/Marti/api/tls/signClient", getSignHandler(app))
	certApi.POST("/Marti/api/tls/signClient/v2", getSignHandlerV2(app))
	certApi.GET("/Marti/api/tls/profile/enrollment", getProfileEnrollmentHandler(app))

	certApi.NotFoundHandler = getNotFoundHandler(app, "cert")

	return certApi
}

func getTlsConfigHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)

		names := map[string]string{"C": "RU"}
		buf := strings.Builder{}
		buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
		buf.WriteString(fmt.Sprintf("<certificateConfig validityDays=\"%d\"><nameEntries>", app.config.certTtlDays))
		for k, v := range names {
			buf.WriteString(fmt.Sprintf("<nameEntry name=\"%s\" value=\"%s\"/>", k, v))
		}
		buf.WriteString("</nameEntries></certificateConfig>")

		res.Header.Set("Content-Type", "application/xml")
		return res.Write(strings.NewReader(buf.String()))
	}
}

func signClientCert(clientCSR *x509.CertificateRequest, serverCert *x509.Certificate, privateKey crypto.PrivateKey, days int) (*x509.Certificate, error) {
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := x509.Certificate{
		Signature:          clientCSR.Signature,
		SignatureAlgorithm: clientCSR.SignatureAlgorithm,

		PublicKeyAlgorithm: clientCSR.PublicKeyAlgorithm,
		PublicKey:          clientCSR.PublicKey,

		SerialNumber: serialNumber,
		Issuer:       serverCert.Subject,
		Subject:      clientCSR.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Duration(days*24) * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, serverCert, clientCSR.PublicKey, privateKey)

	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate, error: %s", err)
	}

	return x509.ParseCertificate(certBytes)
}

func (app *App) processSignRequest(req *air.Request) (*x509.Certificate, error) {
	user := getUsernameFromReq(req)
	uid := getStringParamIgnoreCaps(req, "clientUid")
	ver := getStringParam(req, "version")

	app.Logger.Infof("cert sign req from %s %s ver %s", user, uid, ver)
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	clientCSR, err := tlsutil.ParseCsr(b)
	if err != nil {
		return nil, fmt.Errorf("empty csr block")
	}

	if user != clientCSR.Subject.CommonName {
		return nil, fmt.Errorf("bad user in csr")
	}

	if !app.users.UserIsValid(user, "") {
		return nil, fmt.Errorf("bad user")
	}

	signedCert, err := signClientCert(clientCSR,
		app.config.serverCert, app.config.tlsCert.PrivateKey, app.config.certTtlDays)
	if err != nil {
		return nil, err
	}

	app.onNewCertCreated(user, uid, ver, signedCert.SerialNumber.String())

	return signedCert, nil
}

func getSignHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)

		signedCert, err := app.processSignRequest(req)
		if err != nil {
			app.Logger.Errorf(err.Error())
			return err
		}

		certs := map[string]*x509.Certificate{"signedCert": signedCert}
		for i, c := range app.config.ca {
			certs[fmt.Sprintf("ca%d", i)] = c
		}

		p12Bytes, err := tlsutil.MakeP12TrustStore(certs, p12Password)

		if err != nil {
			app.Logger.Errorf("error making p12: %v", err)
			return err
		}

		return res.Write(bytes.NewReader(p12Bytes))
	}
}

func getSignHandlerV2(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)

		signedCert, err := app.processSignRequest(req)
		if err != nil {
			app.Logger.Errorf(err.Error())
			return err
		}

		accept := req.Header.Get("Accept")
		switch {
		case accept == "", strings.Contains(accept, "*/*"), strings.Contains(accept, "application/json"):
			certs := map[string]string{"signedCert": tlsutil.CertToStr(signedCert, false)}
			for i, c := range app.config.ca {
				certs[fmt.Sprintf("ca%d", i)] = tlsutil.CertToStr(c, false)
			}
			return res.WriteJSON(certs)
		case strings.Contains(accept, "application/xml"):
			buf := strings.Builder{}
			buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
			buf.WriteString("<enrollment>")
			buf.WriteString("<signedCert>")
			buf.WriteString(tlsutil.CertToStr(signedCert, false))
			buf.WriteString("</signedCert>")
			for _, c := range app.config.ca {
				buf.WriteString("<ca>")
				buf.WriteString(tlsutil.CertToStr(c, false))
				buf.WriteString("</ca>")
			}
			buf.WriteString("</enrollment>")
			res.Header.Set("Content-Type", "application/xml; charset=utf-8")
			return res.Write(strings.NewReader(buf.String()))
		default:
			res.Status = http.StatusBadRequest
			return res.WriteString("")
		}
	}
}

func getProfileEnrollmentHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		user := getUsernameFromReq(req)
		uid := getStringParamIgnoreCaps(req, "clientUid")
		app.Logger.Infof("%s %s %s %s", req.Method, req.Path, user, uid)
		files := app.GetProfileFiles(user, uid)
		if len(files) == 0 {
			res.Status = http.StatusNoContent
			return nil
		}

		mp := NewMissionPackage("ProfileMissionPackage-"+uuid.New().String(), "Enrollment")
		mp.Param("onReceiveImport", "true")
		mp.Param("onReceiveDelete", "true")
		for _, f := range files {
			mp.AddFile(f)
		}

		res.Header.Set("Content-Type", "application/zip")
		res.Header.Set("Content-Disposition", "attachment; filename=profile.zip")
		dat, err := mp.Create()
		if err != nil {
			return err
		}
		return res.Write(bytes.NewReader(dat))
	}
}

func (app *App) onNewCertCreated(user, uid, version, serial string) {
	app.Logger.Infof("new cert signed for user %s uid %s ver %s serial %s", user, uid, version, serial)
}
