package main

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/kdudkov/goatak/pkg/log"

	"github.com/kdudkov/goatak/cmd/goatak_server/mp"
	"github.com/kdudkov/goatak/pkg/tlsutil"
)

const (
	p12Password = "atakatak"
)

type CertAPI struct {
	f    *fiber.App
	addr string
}

func NewCertAPI(app *App, addr string) *CertAPI {
	api := &CertAPI{
		f:    fiber.New(fiber.Config{EnablePrintRoutes: false, DisableStartupMessage: true}),
		addr: addr,
	}

	api.f.Use(NewMetricHandler("cert_api"))
	api.f.Use(log.NewFiberLogger(&log.LoggerConfig{Name: "cert_api", UserGetter: Username}))

	api.f.Use(UserAuthHandler(app.users))

	api.f.Get("/Marti/api/tls/config", getTLSConfigHandler(app))
	api.f.Post("/Marti/api/tls/signClient", getSignHandler(app))
	api.f.Post("/Marti/api/tls/signClient/v2", getSignHandlerV2(app))
	api.f.Get("/Marti/api/tls/profile/enrollment", getProfileEnrollmentHandler(app))

	return api
}

func (api *CertAPI) Address() string {
	return api.addr
}

func (api *CertAPI) Listen() error {
	return api.f.Listen(api.addr)
}

func getTLSConfigHandler(app *App) fiber.Handler {
	names := map[string]string{"C": "RU", "O": "goatak", "OU": "goatak"}
	buf := strings.Builder{}
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	buf.WriteString(fmt.Sprintf("<certificateConfig validityDays=\"%d\"><nameEntries>", app.config.CertTTLDays()))

	for k, v := range names {
		buf.WriteString(fmt.Sprintf("<nameEntry name=\"%s\" value=\"%s\"/>", k, v))
	}

	buf.WriteString("</nameEntries></certificateConfig>")

	return func(ctx *fiber.Ctx) error {
		ctx.Set(fiber.HeaderContentType, "application/xml")

		return ctx.SendString(buf.String())
	}
}

func signClientCert(uid string, clientCSR *x509.CertificateRequest, serverCert *x509.Certificate, privateKey crypto.PrivateKey, days int) (*x509.Certificate, error) {
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := x509.Certificate{
		Signature:          clientCSR.Signature,
		SignatureAlgorithm: clientCSR.SignatureAlgorithm,

		PublicKeyAlgorithm: clientCSR.PublicKeyAlgorithm,
		PublicKey:          clientCSR.PublicKey,

		SerialNumber:   serialNumber,
		Issuer:         serverCert.Subject,
		Subject:        clientCSR.Subject,
		NotBefore:      time.Now(),
		NotAfter:       time.Now().Add(time.Duration(days*24) * time.Hour),
		KeyUsage:       x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		EmailAddresses: []string{uid},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, serverCert, clientCSR.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate, error: %w", err)
	}

	return x509.ParseCertificate(certBytes)
}

func (app *App) processSignRequest(ctx *fiber.Ctx) (*x509.Certificate, error) {
	username := Username(ctx)
	uid := ctx.Query("clientUid")
	ver := ctx.Query("version")

	app.logger.Info(fmt.Sprintf("cert sign req from %s %s ver %s", username, uid, ver))

	if !app.users.UserIsValid(username, "") {
		return nil, fmt.Errorf("bad user")
	}

	clientCSR, err := tlsutil.ParseCsr(ctx.Body())
	if err != nil {
		return nil, fmt.Errorf("empty csr block")
	}

	if username != clientCSR.Subject.CommonName {
		return nil, fmt.Errorf("bad user in csr")
	}

	signedCert, err := signClientCert(uid, clientCSR,
		app.config.serverCert, app.config.tlsCert.PrivateKey, app.config.CertTTLDays())
	if err != nil {
		return nil, err
	}

	app.onNewCertCreated(username, uid, ver, signedCert.SerialNumber.String())

	return signedCert, nil
}

func getSignHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		uid := ctx.Query("clientUid")

		if !app.checkUID(uid) {
			app.logger.Warn("blacklisted uid - " + uid)
			return ctx.SendStatus(fiber.StatusForbidden)
		}

		signedCert, err := app.processSignRequest(ctx)
		if err != nil {
			app.logger.Error("error", slog.Any("error", err))

			return err
		}

		certs := map[string]*x509.Certificate{"signedCert": signedCert}
		for i, c := range app.config.ca {
			certs[fmt.Sprintf("ca%d", i)] = c
		}

		p12Bytes, err := tlsutil.MakeP12TrustStore(certs, p12Password)
		if err != nil {
			app.logger.Error("error making p12", slog.Any("error", err))

			return err
		}

		return ctx.Send(p12Bytes)
	}
}

func getSignHandlerV2(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		uid := ctx.Query("clientUid")

		if !app.checkUID(uid) {
			app.logger.Warn("blacklisted uid - " + uid)
			return ctx.SendStatus(fiber.StatusForbidden)
		}

		signedCert, err := app.processSignRequest(ctx)
		if err != nil {
			app.logger.Error("error", slog.Any("error", err))

			return err
		}

		accept := ctx.Get(fiber.HeaderAccept)

		switch {
		case accept == "", strings.Contains(accept, "*/*"), strings.Contains(accept, "application/json"):
			certs := map[string]string{"signedCert": tlsutil.CertToStr(signedCert, false)}
			// iTAK needs server cert in answer
			certs["ca0"] = tlsutil.CertToStr(app.config.serverCert, false)
			for i, c := range app.config.ca {
				certs[fmt.Sprintf("ca%d", i+1)] = tlsutil.CertToStr(c, false)
			}

			return ctx.JSON(certs)
		case strings.Contains(accept, "application/xml"):
			buf := strings.Builder{}
			buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
			buf.WriteString("<enrollment>")
			buf.WriteString("<signedCert>")
			buf.WriteString(tlsutil.CertToStr(signedCert, false))
			buf.WriteString("</signedCert>")
			buf.WriteString("<ca>")
			buf.WriteString(tlsutil.CertToStr(app.config.serverCert, false))
			buf.WriteString("</ca>")

			for _, c := range app.config.ca {
				buf.WriteString("<ca>")
				buf.WriteString(tlsutil.CertToStr(c, false))
				buf.WriteString("</ca>")
			}

			buf.WriteString("</enrollment>")
			ctx.Set(fiber.HeaderContentType, "application/xml; charset=utf-8")

			return ctx.SendString(buf.String())
		default:
			return ctx.SendStatus(fiber.StatusBadRequest)
		}
	}
}

func getProfileEnrollmentHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		username := Username(ctx)
		uid := ctx.Query("clientUid")

		if !app.checkUID(uid) {
			app.logger.Warn("blacklisted uid - " + uid)
			return ctx.SendStatus(fiber.StatusForbidden)
		}

		files := app.GetProfileFiles(username, uid)
		if len(files) == 0 {
			return ctx.SendStatus(fiber.StatusNoContent)
		}

		pkg := mp.NewMissionPackage("ProfileMissionPackage-"+uuid.NewString(), "Enrollment")
		pkg.Param("onReceiveImport", "true")
		pkg.Param("onReceiveDelete", "true")

		for _, f := range files {
			pkg.AddFile(f)
		}

		ctx.Set(fiber.HeaderContentType, "application/zip")
		ctx.Set(fiber.HeaderContentDisposition, "attachment; filename=profile.zip")

		dat, err := pkg.Create()
		if err != nil {
			return err
		}

		return ctx.Send(dat)
	}
}

func (app *App) onNewCertCreated(user, uid, version, serial string) {
	app.logger.Info(fmt.Sprintf("new cert signed for user %s uid %s ver %s serial %s", user, uid, version, serial))
}
