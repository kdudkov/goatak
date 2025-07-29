package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/kdudkov/goatak/internal/client"
	"github.com/kdudkov/goatak/pkg/tlsutil"
)

func (app *App) ListenTCP(ctx context.Context, addr string) (err error) {
	app.logger.Info("listening TCP at " + addr)
	defer func() {
		if r := recover(); r != nil {
			app.logger.Error("panic in ListenTCP", slog.Any("error", r))
		}
	}()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		app.logger.Error("Failed to listen", slog.Any("error", err))

		return err
	}

	defer listener.Close()

	for ctx.Err() == nil {
		conn, err := listener.Accept()
		if err != nil {
			app.logger.Error("Unable to accept connections", slog.Any("error", err))

			return err
		}

		app.logger.Info("TCP connection from" + conn.RemoteAddr().String())
		name := "tcp:" + conn.RemoteAddr().String()
		h := client.NewConnClientHandler(name, conn, &client.HandlerConfig{
			MessageCb:    app.NewCotMessage,
			RemoveCb:     app.RemoveHandlerCb,
			NewContactCb: app.NewContactCb,
			DropMetric:   dropMetric,
			UidChecker:   app.checkUID,
		})
		app.AddClientHandler(h)
		h.Start()
	}

	return nil
}

func (app *App) listenTLS(ctx context.Context, addr string) error {
	app.logger.Info("listening TCP SSL at " + addr)

	defer func() {
		if r := recover(); r != nil {
			app.logger.Error("panic in ListenTLS", slog.Any("error", r))
		}
	}()

	tlsCfg := &tls.Config{
		Certificates:     []tls.Certificate{*app.config.TlsCert},
		ClientCAs:        app.config.CertPool,
		ClientAuth:       tls.RequireAndVerifyClientCert,
		VerifyConnection: app.verifyConnection,
	}

	listener, err := tls.Listen("tcp", addr, tlsCfg)
	if err != nil {
		return err
	}

	defer listener.Close()

	for ctx.Err() == nil {
		conn, err := listener.Accept()
		if err != nil {
			app.logger.Error("Unable to accept connections", slog.Any("error", err))

			continue
		}

		app.logger.Debug("SSL connection from " + conn.RemoteAddr().String())

		go app.processTLSConn(ctx, conn.(*tls.Conn))
	}

	return nil
}

func (app *App) processTLSConn(ctx context.Context, conn *tls.Conn) {
	ctx1, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	if err := conn.HandshakeContext(ctx1); err != nil {
		app.logger.Debug("Handshake error", slog.Any("error", err))
		_ = conn.Close()

		return
	}

	st := conn.ConnectionState()
	username, sn := getCertUser(&st)
	uid := getCertUID(&st)
	if !app.users.IsValid(username, sn) {
		app.logger.Info(fmt.Sprintf("bad user %s, sn %s", username, sn))
		return
	}

	app.users.SaveConnectInfo(username, uid, sn)

	name := "ssl:" + conn.RemoteAddr().String()
	h := client.NewConnClientHandler(name, conn, &client.HandlerConfig{
		Device:       app.users.Get(username),
		Serial:       sn,
		MessageCb:    app.NewCotMessage,
		RemoveCb:     app.RemoveHandlerCb,
		NewContactCb: app.NewContactCb,
		DropMetric:   dropMetric,
		UidChecker:   app.checkUID,
	})
	app.AddClientHandler(h)
	h.Start()
}

func (app *App) verifyConnection(st tls.ConnectionState) error {
	user, sn := getCertUser(&st)
	tlsutil.LogCerts(app.logger, st.PeerCertificates...)

	if !app.users.IsValid(user, sn) {
		app.logger.Warn("bad user " + user)

		return fmt.Errorf("bad user")
	}

	return nil
}

func getCert(st *tls.ConnectionState) *x509.Certificate {
	for _, cert := range st.PeerCertificates {
		if cert.Subject.CommonName != "" {
			return cert
		}
	}

	return nil
}

func getCertUser(st *tls.ConnectionState) (string, string) {
	if cert := getCert(st); cert != nil {
		return cert.Subject.CommonName, hex.EncodeToString(cert.SerialNumber.Bytes())
	}

	return "", ""
}

func getCertUID(st *tls.ConnectionState) string {
	if cert := getCert(st); cert != nil && len(cert.EmailAddresses) > 0 {
		return cert.EmailAddresses[0]
	}

	return ""
}
