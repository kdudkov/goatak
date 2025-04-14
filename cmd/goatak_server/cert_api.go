package main

import (
	"crypto"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
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
	tls  bool
	cert tls.Certificate
}

func NewCertAPI(app *App, addr string) *CertAPI {
	api := &CertAPI{
		f:    fiber.New(fiber.Config{EnablePrintRoutes: false, DisableStartupMessage: true}),
		addr: addr,
	}

	api.f.Use(NewMetricHandler("cert_api"))
	api.f.Use(log.NewFiberLogger(&log.LoggerConfig{Name: "cert_api", UserGetter: Username}))

	api.f.Use(UserAuthHandler(app.users))

	if app.config.EnrollSSL() {
		api.tls = true
		api.cert = *app.config.TlsCert
	}

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
	if api.tls {
		return api.f.ListenTLSWithCertificate(api.addr, api.cert)
	} else {
		return api.f.Listen(api.addr)
	}
}

func getTLSConfigHandler(app *App) fiber.Handler {
	names := map[string]string{"C": "RU", "ST": "RU", "L": "SPB", "O": "goatak", "OU": "goatak"}
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
	tpl := getCertTemplate(&serverCert.Subject, clientCSR, uid, days)

	certBytes, err := x509.CreateCertificate(rand.Reader, tpl, serverCert, clientCSR.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate, error: %w", err)
	}

	return x509.ParseCertificate(certBytes)
}

func getCertTemplate(issuer *pkix.Name, csr *x509.CertificateRequest, uid string, days int) *x509.Certificate {
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	return &x509.Certificate{
		Signature:          csr.Signature,
		SignatureAlgorithm: csr.SignatureAlgorithm,

		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		PublicKey:          csr.PublicKey,

		SerialNumber:   serialNumber,
		Issuer:         *issuer,
		Subject:        csr.Subject,
		NotBefore:      time.Now(),
		NotAfter:       time.Now().Add(time.Duration(days*24) * time.Hour),
		KeyUsage:       x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		EmailAddresses: []string{uid},
	}
}

func (app *App) processSignRequest(ctx *fiber.Ctx) (*x509.Certificate, error) {
	username := Username(ctx)
	uid := queryIgnoreCase(ctx, "clientUid")
	ver := ctx.Query("version")

	app.logger.Info(fmt.Sprintf("cert sign req from %s %s ver %s", username, uid, ver))

	if !app.users.IsValid(username, "") {
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
		app.config.ServerCert, app.config.TlsCert.PrivateKey, app.config.CertTTLDays())
	if err != nil {
		return nil, err
	}

	serial := signedCert.SerialNumber.String()
	app.users.SaveSignInfo(username, uid, serial)
	app.logger.Info(fmt.Sprintf("new cert signed for user %s uid %s ver %s serial %s", username, uid, ver, serial))

	return signedCert, nil
}

func getSignHandler(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		uid := queryIgnoreCase(ctx, "clientUid")

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
		certs["ca0"] = app.config.ServerCert
		for i, c := range app.config.CA {
			certs[fmt.Sprintf("ca%d", i+1)] = c
		}

		p12Bytes, err := tlsutil.MakeP12TrustStoreNamed(p12Password, certs)
		if err != nil {
			app.logger.Error("error making p12", slog.Any("error", err))

			return err
		}

		return ctx.Send(p12Bytes)
	}
}

func getSignHandlerV2(app *App) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		uid := queryIgnoreCase(ctx, "clientUid")

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
			certs["ca0"] = tlsutil.CertToStr(app.config.ServerCert, false)
			for i, c := range app.config.CA {
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
			buf.WriteString(tlsutil.CertToStr(app.config.ServerCert, false))
			buf.WriteString("</ca>")

			for _, c := range app.config.CA {
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
		uid := queryIgnoreCase(ctx, "clientUid")

		if !app.checkUID(uid) {
			app.logger.Warn("blacklisted uid - " + uid)
			return ctx.SendStatus(fiber.StatusForbidden)
		}

		files := app.GetProfileFiles(username, uid)
		if len(files) == 0 {
			return ctx.SendStatus(fiber.StatusNoContent)
		}

		pkg := mp.NewMissionPackage(uuid.NewSHA1(uuid.Nil, []byte(username)).String(), "Enrollment")
		pkg.Param("onReceiveDelete", "true")

		for _, f := range files {
			pkg.AddFile(f)
		}

		dat, err := pkg.Create()
		if err != nil {
			return err
		}

		ctx.Set(fiber.HeaderContentType, "application/zip")
		ctx.Set(fiber.HeaderContentDisposition, "attachment; filename=profile.zip")
		return ctx.Send(dat)
	}
}
