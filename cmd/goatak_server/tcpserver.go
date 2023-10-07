package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
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
			Logger:    app.Logger.With(zap.String("addr", name)),
			MessageCb: app.NewCotMessage,
			RemoveCb:  app.RemoveHandlerCb})
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
		app.Logger.Infof("SSL connection from %s", conn.RemoteAddr())
		c1 := conn.(*tls.Conn)
		if err := c1.Handshake(); err != nil {
			app.Logger.Errorf("Handshake error: %#v", err)
			c1.Close()
			continue
		}

		st := c1.ConnectionState()
		username, serial := getCertUser(&st)
		var scope string
		if user := app.users.GetUser(username); user != nil {
			scope = user.Scope
		}

		name := "ssl:" + conn.RemoteAddr().String()
		h := client.NewConnClientHandler(name, conn, &client.HandlerConfig{
			Logger:    app.Logger.With(zap.String("user", username), zap.String("addr", name)),
			User:      username,
			Scope:     scope,
			Serial:    serial,
			MessageCb: app.NewCotMessage,
			RemoveCb:  app.RemoveHandlerCb})
		app.AddClientHandler(h)
		h.Start()
		app.onTlsClientConnect(username, serial)
	}
}

func (app *App) verifyConnection(st tls.ConnectionState) error {
	user, sn := getCertUser(&st)
	app.logCert(st.PeerCertificates)

	if !app.users.UserIsValid(user, sn) {
		app.Logger.Warnf("bad user %s", user)
		return fmt.Errorf("bad user %s", user)
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

func (app *App) logCert(cert []*x509.Certificate) {
	for i, cert := range cert {
		app.Logger.Infof("#%d issuer: %s", i, cert.Issuer.String())
		app.Logger.Infof("#%d subject: %s", i, cert.Subject.String())
		app.Logger.Infof("#%d sn: %x", i, cert.SerialNumber)
	}
}

func (app *App) onTlsClientConnect(username, sn string) {

}
