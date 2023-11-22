package main

import (
	"crypto/tls"
	"fmt"
	"github.com/kdudkov/goatak/pkg/tlsutil"
	"net"

	"go.uber.org/zap"

	"github.com/kdudkov/goatak/internal/client"
)

func (app *App) ListenTCP(addr string) (err error) {
	app.Logger.Infof("listening TCP at %s", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		app.Logger.Errorf("Failed to listen: %v", err)
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			app.Logger.Errorf("Unable to accept connections: %#v", err)
			return err
		}
		app.Logger.Infof("TCP connection from %s", conn.RemoteAddr())
		name := "tcp:" + conn.RemoteAddr().String()
		h := client.NewConnClientHandler(name, conn, &client.HandlerConfig{
			Logger:       app.Logger.With(zap.String("addr", name)),
			MessageCb:    app.NewCotMessage,
			RemoveCb:     app.RemoveHandlerCb,
			NewContactCb: app.NewContactCb})
		app.AddClientHandler(h)
		h.Start()
	}
}

func (app *App) listenTls(addr string) error {
	app.Logger.Infof("listening TCP SSL at %s", addr)

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

	for {
		conn, err := listener.Accept()
		if err != nil {
			app.Logger.Errorf("Unable to accept connections: %#v", err)
			continue
		}
		app.Logger.Debugf("SSL connection from %s", conn.RemoteAddr())
		c1 := conn.(*tls.Conn)
		if err := c1.Handshake(); err != nil {
			app.Logger.Debugf("Handshake error: %#v", err)
			c1.Close()
			continue
		}

		st := c1.ConnectionState()
		username, serial := getCertUser(&st)

		name := "ssl:" + conn.RemoteAddr().String()
		h := client.NewConnClientHandler(name, conn, &client.HandlerConfig{
			Logger:       app.Logger.With(zap.String("user", username), zap.String("addr", name)),
			User:         app.users.GetUser(username),
			Serial:       serial,
			MessageCb:    app.NewCotMessage,
			RemoveCb:     app.RemoveHandlerCb,
			NewContactCb: app.NewContactCb})
		app.AddClientHandler(h)
		h.Start()
		app.onTlsClientConnect(username, serial)
	}
}

func (app *App) verifyConnection(st tls.ConnectionState) error {
	user, sn := getCertUser(&st)
	tlsutil.LogCerts(app.Logger, st.PeerCertificates...)

	if !app.users.UserIsValid(user, sn) {
		app.Logger.Warnf("bad user %s", user)
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

func (app *App) onTlsClientConnect(username, sn string) {

}
