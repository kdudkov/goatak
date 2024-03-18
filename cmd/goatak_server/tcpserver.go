package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/spf13/viper"
	"net"

	"github.com/kdudkov/goatak/internal/client"
	"github.com/kdudkov/goatak/pkg/tlsutil"
)

func (app *App) ListenTCP(ctx context.Context, addr string) (err error) {
	app.logger.Info("listening TCP at " + addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		app.logger.Error("Failed to listen", "error", err)

		return err
	}

	defer listener.Close()

	for ctx.Err() == nil {
		conn, err := listener.Accept()
		if err != nil {
			app.logger.Error("Unable to accept connections", "error", err)

			return err
		}

		app.logger.Info("TCP connection from" + conn.RemoteAddr().String())
		name := "tcp:" + conn.RemoteAddr().String()
		h := client.NewConnClientHandler(name, conn, &client.HandlerConfig{
			MessageCb:    app.NewCotMessage,
			RemoveCb:     app.RemoveHandlerCb,
			NewContactCb: app.NewContactCb,
			RoutePings:   viper.GetBool("route_pings"),
		})
		app.AddClientHandler(h)
		h.Start()
	}

	return nil
}

func (app *App) listenTLS(ctx context.Context, addr string) error {
	app.logger.Info("listening TCP SSL at " + addr)

	tlsCfg := &tls.Config{
		Certificates:     []tls.Certificate{*app.config.tlsCert},
		ClientCAs:        app.config.certPool,
		ClientAuth:       tls.RequireAndVerifyClientCert,
		VerifyConnection: app.verifyConnection,
	}

	listener, err := tls.Listen("tcp4", addr, tlsCfg)
	if err != nil {
		return err
	}

	defer listener.Close()

	for ctx.Err() == nil {
		conn, err := listener.Accept()
		if err != nil {
			app.logger.Error("Unable to accept connections", "error", err)

			continue
		}

		app.logger.Debug("SSL connection from " + conn.RemoteAddr().String())

		c1 := conn.(*tls.Conn)
		if err := c1.Handshake(); err != nil {
			app.logger.Debug("Handshake error", "error", err)
			c1.Close()

			continue
		}

		st := c1.ConnectionState()
		username, serial := getCertUser(&st)

		name := "ssl:" + conn.RemoteAddr().String()
		h := client.NewConnClientHandler(name, conn, &client.HandlerConfig{
			User:         app.users.GetUser(username),
			Serial:       serial,
			MessageCb:    app.NewCotMessage,
			RemoveCb:     app.RemoveHandlerCb,
			NewContactCb: app.NewContactCb,
			RoutePings:   viper.GetBool("route_pings"),
		})
		app.AddClientHandler(h)
		h.Start()
		app.onTLSClientConnect(username, serial)
	}

	return nil
}

func (app *App) verifyConnection(st tls.ConnectionState) error {
	user, sn := getCertUser(&st)
	tlsutil.LogCerts(app.logger, st.PeerCertificates...)

	if !app.users.UserIsValid(user, sn) {
		app.logger.Warn("bad user " + user)

		return fmt.Errorf("bad user")
	}

	return nil
}

func getCertUser(st *tls.ConnectionState) (string, string) {
	for _, cert := range st.PeerCertificates {
		if cert.Subject.CommonName != "" {
			return cert.Subject.CommonName, fmt.Sprintf("%x", cert.SerialNumber)
		}
	}

	return "", ""
}

func (app *App) onTLSClientConnect(username, sn string) {
	//no-op
}
