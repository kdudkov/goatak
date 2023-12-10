package main

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/spf13/viper"

	"github.com/kdudkov/goatak/pkg/tlsutil"
)

func (app *App) connect() (net.Conn, error) {
	addr := fmt.Sprintf("%s:%s", app.host, app.tcpPort)
	if app.tls {
		app.Logger.Infof("connecting with SSL to %s...", addr)

		conn, err := tls.Dial("tcp", addr, app.getTLSConfig())
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
		tlsutil.LogCerts(app.Logger, cs.PeerCertificates...)

		return conn, nil
	}
	app.Logger.Infof("connecting to %s...", addr)

	return net.DialTimeout("tcp", addr, app.dialTimeout)
}

func (app *App) getTLSConfig() *tls.Config {
	conf := &tls.Config{ //nolint:exhaustruct
		Certificates: []tls.Certificate{*app.tlsCert},
		RootCAs:      app.cas,
		ClientCAs:    app.cas,
	}

	if !viper.GetBool("ssl.strict") {
		conf.InsecureSkipVerify = true
	}

	return conf
}
