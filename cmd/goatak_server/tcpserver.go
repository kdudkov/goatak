package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"

	"github.com/kdudkov/goatak/cot"
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
		h := cot.NewClientHandler(name, conn, &cot.HandlerConfig{
			Logger:    app.Logger,
			MessageCb: app.NewCotMessage,
			RemoveCb:  app.RemoveHandlerCb})
		app.handlers.Store(name, h)
		h.Start()
	}
}

func (app *App) ListenSSl(addr string) error {
	app.Logger.Infof("listening TCP SSL at %s", addr)

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{*app.config.cert},
		ClientCAs:    app.config.ca,
		ClientAuth:   tls.RequireAndVerifyClientCert,
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

		app.logCert(c1.ConnectionState().PeerCertificates)
		user, serial := getUser(c1)
		app.Logger.Infof("user: %s, sn: %s", user, serial)
		name := "ssl:" + conn.RemoteAddr().String()
		h := cot.NewClientHandler(name, conn, &cot.HandlerConfig{
			Logger:    app.Logger,
			User:      user,
			MessageCb: app.NewCotMessage,
			RemoveCb:  app.RemoveHandlerCb})
		app.handlers.Store(name, h)
		h.Start()
	}
}

func getUser(conn *tls.Conn) (string, string) {
	for _, cert := range conn.ConnectionState().PeerCertificates {
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
