package main

import (
	"encoding/xml"
	"net"

	"github.com/kdudkov/goatak/cot"
	"github.com/kdudkov/goatak/cotxml"
)

func (app *App) ListenUDP(addr string) error {
	app.Logger.Infof("listening UDP at %s", addr)
	p, err := net.ListenPacket("udp", addr)

	if err != nil {
		app.Logger.Error(err)
		return err
	}

	buf := make([]byte, 65535)

	for app.ctx.Err() == nil {
		n, _, err := p.ReadFrom(buf)
		if err != nil {
			app.Logger.Errorf("read error: %v", err)
			return err
		}

		evt := &cotxml.Event{}
		if err := xml.Unmarshal(buf[:n], evt); err != nil {
			app.Logger.Errorf("decode error: %v", err)
			continue
		}

		msg, xd := cot.EventToProto(evt)

		app.NewCotMessage(&cot.Msg{
			TakMessage: msg,
			Detail:     xd,
		})
	}

	return nil
}
