package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/air-gases/authenticator"
	"github.com/aofei/air"
	"software.sslmate.com/src/go-pkcs12"
)

const certTtl = time.Hour * 24 * 60

func getCertApi(app *App, addr string) *air.Air {
	certApi := air.New()
	certApi.Address = addr

	auth := authenticator.BasicAuthGas(authenticator.BasicAuthGasConfig{
		Validator: func(username string, password string, _ *air.Request, _ *air.Response) (bool, error) {
			app.Logger.Infof("tls api login with user %s", username)
			return app.CheckUserAuth(username, password), nil
		},
	})

	certApi.Gases = []air.Gas{auth}

	certApi.GET("/Marti/api/tls/config", getTlsConfigHandler(app))
	certApi.POST("/Marti/api/tls/signClient", getSignHandler(app))
	certApi.POST("/Marti/api/tls/signClient/v2", getSignHandlerV2(app))
	certApi.GET("/Marti/api/tls/profile/enrollment", getProfileEnrollmentHandler(app))

	certApi.NotFoundHandler = getNotFoundHandler(app)

	if app.config.useSsl {
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{*app.config.tlsCert},
			ClientCAs:    app.config.certPool,
			RootCAs:      app.config.certPool,
			ClientAuth:   tls.NoClientCert,
			MinVersion:   tls.VersionTLS10,
		}

		certApi.TLSConfig = tlsCfg
	}
	return certApi
}

func getTlsConfigHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)

		s := strings.Builder{}
		s.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
		s.WriteString("<certificateConfig><nameEntries>")
		s.WriteString("<nameEntry name=\"C\" value=\"RU\"/>")
		s.WriteString("</nameEntries></certificateConfig>")

		return res.WriteString(s.String())
	}
}

func signClientCert(clientCSR *x509.CertificateRequest, serverCert *x509.Certificate, privateKey crypto.PrivateKey) (*x509.Certificate, error) {
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := x509.Certificate{
		Signature:          clientCSR.Signature,
		SignatureAlgorithm: clientCSR.SignatureAlgorithm,

		PublicKeyAlgorithm: clientCSR.PublicKeyAlgorithm,
		PublicKey:          clientCSR.PublicKey,

		SerialNumber: serialNumber,
		Issuer:       serverCert.Subject,
		Subject:      clientCSR.Subject,
		NotBefore:    time.Now().Add(-5 * time.Minute),
		NotAfter:     time.Now().Add(certTtl),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, serverCert, clientCSR.PublicKey, privateKey)

	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate, error: %s", err)
	}

	return x509.ParseCertificate(certBytes)
}

func parseCsr(b []byte) (*x509.CertificateRequest, error) {
	bb := bytes.Buffer{}
	bb.WriteString("-----BEGIN CERTIFICATE REQUEST-----\n")
	bb.Write(b)
	bb.WriteString("\n-----END CERTIFICATE REQUEST-----")
	csrBlock, _ := pem.Decode(bb.Bytes())

	return x509.ParseCertificateRequest(csrBlock.Bytes)
}

func makeP12(certs map[string]*x509.Certificate) ([]byte, error) {
	var entries []pkcs12.TrustStoreEntry

	for k, v := range certs {
		entries = append(entries, pkcs12.TrustStoreEntry{Cert: v, FriendlyName: k})
	}

	return pkcs12.EncodeTrustStoreEntries(rand.Reader, entries, "atakatak")
}

func certToPem(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
}

func getSignHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)

		uid := getStringParamIgnoreCaps(req, "clientUid")
		ver := getStringParam(req, "version")

		app.Logger.Infof("cert sign req from %s ver %s", uid, ver)
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}

		clientCSR, err := parseCsr(b)
		if err != nil {
			app.Logger.Warnf("empry csr block")
			return fmt.Errorf("empty csr block")
		}

		if !app.UserIsValid(clientCSR.Subject.CommonName) {
			app.Logger.Warnf("bad user %s", clientCSR.Subject.CommonName)
			return fmt.Errorf("bad user")
		}

		signedCert, err := signClientCert(clientCSR, app.config.cert, app.config.tlsCert.PrivateKey)
		if err != nil {
			app.Logger.Errorf("error signing cert: %v", err)
			return err
		}

		certs := map[string]*x509.Certificate{"signedCert": signedCert}
		for i, c := range app.config.ca {
			certs[fmt.Sprintf("ca%d", i)] = c
		}

		p12Bytes, err := makeP12(certs)

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

		uid := getStringParamIgnoreCaps(req, "clientUid")
		ver := getStringParam(req, "version")

		app.Logger.Infof("cert sign reqv2 from %s ver %s", uid, ver)
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}

		clientCSR, err := parseCsr(b)
		if err != nil {
			app.Logger.Warnf("empry csr block")
			return fmt.Errorf("empty csr block")
		}

		if !app.UserIsValid(clientCSR.Subject.CommonName) {
			app.Logger.Warnf("bad user %s", clientCSR.Subject.CommonName)
			return fmt.Errorf("bad user")
		}

		signedCert, err := signClientCert(clientCSR, app.config.cert, app.config.tlsCert.PrivateKey)
		if err != nil {
			app.Logger.Errorf("error signing cert: %v", err)
			return err
		}

		accept := req.Header.Get("Accept")
		switch {
		case accept == "", strings.Contains(accept, "*/*"), strings.Contains(accept, "application/json"):
			certs := map[string]string{"signedCert": string(certToPem(signedCert))}
			for i, c := range app.config.ca {
				certs[fmt.Sprintf("ca%d", i)] = string(certToPem(c))
			}
			return res.WriteJSON(certs)
		case strings.Contains(accept, "application/xml"):
			buf := strings.Builder{}
			buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
			buf.WriteString("<enrollment>")
			buf.WriteString("<signedCert>")
			buf.WriteString(string(certToPem(signedCert)))
			buf.WriteString("</signedCert>")
			for _, c := range app.config.ca {
				buf.WriteString("<ca>")
				buf.WriteString(string(certToPem(c)))
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
		app.Logger.Infof("%s %s", req.Method, req.Path)
		uid := getStringParamIgnoreCaps(req, "clientUid")
		app.Logger.Infof("Profile enrollment req from %s", uid)

		return nil
	}
}
