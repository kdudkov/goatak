package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
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
		app.Logger.Infof("TCP connection")
		NewClientHandler(conn, app).Start()
	}
}

func (app *App) ListenSSl(certFile, keyFile, caFile, addr string) error {
	app.Logger.Infof("listening TCP SSL at %s", addr)

	caCertPEM, err := ioutil.ReadFile(caFile)
	if err != nil {
		return err
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(caCertPEM)
	if !ok {
		return fmt.Errorf("failed to parse root certificate")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    roots,
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
		app.Logger.Infof("SSL connection")
		c1 := conn.(*tls.Conn)
		if err := c1.Handshake(); err != nil {
			app.Logger.Errorf("Handshake error: %#v", err)
			c1.Close()
			continue
		}

		user := getUser(c1)

		app.Logger.Infof("user: %s", user)
		NewClientHandler(c1, app).Start()
	}
}

func getUser(conn *tls.Conn) string {
	for _, cert := range conn.ConnectionState().PeerCertificates {
		if cert.Subject.CommonName != "" {
			return cert.Subject.CommonName
		}
	}

	return ""
}
