package main

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/kdudkov/goatak/pkg/tlsutil"
)

func (app *App) connect() (net.Conn, error) {
	addr := fmt.Sprintf("%s:%s", app.host, app.tcpPort)
	if app.tls {
		app.logger.Info(fmt.Sprintf("connecting with SSL to %s...", addr))

		conn, err := tls.Dial("tcp", addr, app.getTLSConfig())
		if err != nil {
			return nil, err
		}

		app.logger.Debug("handshake...")

		if err := conn.Handshake(); err != nil {
			return conn, err
		}

		cs := conn.ConnectionState()

		app.logger.Info(fmt.Sprintf("Handshake complete: %t", cs.HandshakeComplete))
		app.logger.Info(fmt.Sprintf("version: %d", cs.Version))
		tlsutil.LogCerts(app.logger, cs.PeerCertificates...)

		return conn, nil
	}
	app.logger.Info(fmt.Sprintf("connecting to %s...", addr))

	return net.DialTimeout("tcp", addr, app.dialTimeout)
}
