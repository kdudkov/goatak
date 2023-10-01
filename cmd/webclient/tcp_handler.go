package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
)

func (app *App) connect() (net.Conn, error) {
	addr := fmt.Sprintf("%s:%s", app.host, app.tcpPort)
	if app.tls {
		app.Logger.Infof("connecting with SSL to %s...", addr)
		conn, err := tls.Dial("tcp", addr, app.getTlsConfig())
		if err != nil {
			return nil, err
		}
		app.Logger.Debugf("handshake...")

		if err := conn.Handshake(); err != nil {
			return conn, err
		}
		cs := conn.ConnectionState()

		app.Logger.Infof("Handshake complete: %t", cs.HandshakeComplete)
		app.Logger.Infof("version: %d", cs.Version)
		for i, cert := range cs.PeerCertificates {
			app.Logger.Infof("cert #%d subject: %s", i, cert.Subject.String())
			app.Logger.Infof("cert #%d issuer: %s", i, cert.Issuer.String())
			app.Logger.Infof("cert #%d dns_names: %s", i, strings.Join(cert.DNSNames, ","))
		}
		return conn, nil
	} else {
		app.Logger.Infof("connecting to %s...", addr)
		return net.DialTimeout("tcp", addr, app.dialTimeout)
	}
}

func (app *App) getTlsConfig() *tls.Config {
	return &tls.Config{Certificates: []tls.Certificate{*app.tlsCert}, InsecureSkipVerify: true}
}
