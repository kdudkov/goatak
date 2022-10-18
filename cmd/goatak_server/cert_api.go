package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/air-gases/authenticator"
	"github.com/aofei/air"
	"software.sslmate.com/src/go-pkcs12"
)

func getCertApi(app *App, addr string) *air.Air {
	certApi := air.New()
	certApi.Address = addr

	if app.config.useSsl {
		_ = authenticator.BasicAuthGas(authenticator.BasicAuthGasConfig{
			Validator: func(username string, password string, _ *air.Request, _ *air.Response) (bool, error) {
				app.Logger.Infof("tls api login with user %s", username)
				return username == "user", nil
			},
		})

		//certApi.Gases = []air.Gas{auth}
	}

	certApi.GET("/Marti/api/tls/config", getTlsConfigHandler(app))
	certApi.POST("/Marti/api/tls/signClient", getSignHandler(app))
	certApi.GET("/Marti/api/tls/profile/enrollment", getProfileEnrollmentHandler(app))

	certApi.NotFoundHandler = getNotFoundHandler(app)

	if app.config.useSsl {
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{*app.config.tlsCert},
			ClientCAs:    app.config.ca,
			RootCAs:      app.config.ca,
			ClientAuth:   tls.NoClientCert,
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
		//s.WriteString("<nameEntry name=\"name\" value=\"value\"/>")
		s.WriteString("</nameEntries></certificateConfig>")

		return res.WriteString(s.String())
	}
}

func getSignHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)

		uid := getStringParam(req, "clientUid")
		ver := getStringParam(req, "version")

		app.Logger.Infof("cert sign req from %s ver %s", uid, ver)
		b, err := io.ReadAll(req.Body)

		bb := bytes.Buffer{}
		bb.WriteString("-----BEGIN CERTIFICATE REQUEST-----")
		bb.Write(b)
		bb.WriteString("-----END CERTIFICATE REQUEST-----")
		csrBlock, _ := pem.Decode(bb.Bytes())

		if csrBlock == nil {
			return fmt.Errorf("error")
		}

		clientCSR, err := x509.ParseCertificateRequest(csrBlock.Bytes)
		if err != nil {
			return err
		}

		if err = clientCSR.CheckSignature(); err != nil {
			return err
		}

		if !app.config.UserIsValid(clientCSR.Subject.CommonName) {
			app.Logger.Warnf("bad user %s", clientCSR.Subject.CommonName)
			return fmt.Errorf("bad user")
		}

		app.Logger.Infof("makeing cert for %s", clientCSR.Subject.CommonName)

		serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

		template := x509.Certificate{
			Signature:          clientCSR.Signature,
			SignatureAlgorithm: clientCSR.SignatureAlgorithm,

			PublicKeyAlgorithm: clientCSR.PublicKeyAlgorithm,
			PublicKey:          clientCSR.PublicKey,

			SerialNumber: serialNumber,
			Issuer:       app.config.cert.Subject,
			Subject:      clientCSR.Subject,
			NotBefore:    time.Now(),
			NotAfter:     time.Now().Add(365 * 24 * time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}

		certBytes, err := x509.CreateCertificate(rand.Reader, &template, app.config.cert, clientCSR.PublicKey, app.config.tlsCert.PrivateKey)

		if err != nil {
			app.Logger.Errorf("failed to generate certificate, error: %s", err)
			return fmt.Errorf("failed to generate certificate, error: %s", err)
		}

		cert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			app.Logger.Errorf("failed to parse certificate, error: %s", err)
			return fmt.Errorf("failed to generate certificate, error: %s", err)
		}

		entr := []pkcs12.TrustStoreEntry{
			{
				Cert:         cert,
				FriendlyName: "signedCert",
			},
			{
				Cert:         app.config.cert,
				FriendlyName: "server",
			},
		}

		p12Bytes, err := pkcs12.EncodeTrustStoreEntries(rand.Reader, entr, "atakatak")

		if err != nil {
			app.Logger.Errorf("failed to encode truststore, error: %s", err)
			return fmt.Errorf("failed to encode truststore, error: %s", err)
		}

		return res.Write(bytes.NewReader(p12Bytes))
	}
}

func getProfileEnrollmentHandler(app *App) func(req *air.Request, res *air.Response) error {
	return func(req *air.Request, res *air.Response) error {
		app.Logger.Infof("%s %s", req.Method, req.Path)
		uid := getStringParam(req, "clientUid")
		app.Logger.Infof("Profile enrollment req from %s", uid)

		return nil
	}
}
