package main

import (
	"context"
	"crypto"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kdudkov/goatak/cot"
	"github.com/kdudkov/goatak/cotproto"
	"github.com/spf13/viper"
	"software.sslmate.com/src/go-pkcs12"
)

func (app *App) connect() error {
	var err error

	if app.tls {
		app.Logger.Infof("connecting with SSL to %s...", app.addr)
		if app.conn, err = tls.Dial("tcp", app.addr, app.getTlsConfig()); err != nil {
			return err
		}
	} else {
		app.Logger.Infof("connecting to %s...", app.addr)
		if app.conn, err = net.Dial("tcp", app.addr); err != nil {
			return err
		}
	}
	return nil
}

func (app *App) getTlsConfig() *tls.Config {
	p12Data, err := ioutil.ReadFile(viper.GetString("ssl.cert"))
	if err != nil {
		app.Logger.Fatal(err)
	}

	key, cert, _, err := pkcs12.DecodeChain(p12Data, "atakatak")
	if err != nil {
		app.Logger.Fatal(err)
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  key.(crypto.PrivateKey),
		Leaf:        cert,
	}

	return &tls.Config{Certificates: []tls.Certificate{tlsCert}, InsecureSkipVerify: true}
}

func (app *App) reader(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc) {
	defer wg.Done()
	n := 0
	er := cot.NewTagReader(app.conn)
	pr := cot.NewProtoReader(app.conn)
	app.Logger.Infof("start reader")

Loop:
	for ctx.Err() == nil {
		app.conn.SetReadDeadline(time.Now().Add(time.Second * 120))

		var msg *cotproto.TakMessage
		var d *cot.XMLDetails
		var err error

		switch atomic.LoadUint32(&app.ver) {
		case 0:
			msg, d, err = app.processXMLRead(er)
		case 1:
			msg, d, err = app.processProtoRead(pr)
		}

		if err != nil {
			if err == io.EOF {
				break Loop
			}
			app.Logger.Errorf("%v", err)
			break Loop
		}

		if msg == nil {
			continue
		}

		if err != nil {
			app.Logger.Errorf("error decoding details: %v", err)
			return
		}

		app.ProcessEvent(&cot.Msg{
			TakMessage: msg,
			Detail:     d,
		})
		n++
	}

	app.setOnline(false)
	app.conn.Close()
	cancel()
	app.Logger.Infof("got %d messages", n)
}

func (app *App) writer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

Loop:
	for {
		if !app.isOnline() {
			break
		}
		select {
		case msg := <-app.ch:
			app.setWriteActivity()
			if len(msg) == 0 {
				break
			}
			if _, err := app.conn.Write(msg); err != nil {
				break Loop
			}
		case <-ctx.Done():
			break Loop
		}
	}

	app.conn.Close()
}
