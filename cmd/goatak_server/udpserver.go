package main

import (
	"encoding/xml"
	"google.golang.org/protobuf/proto"
	"net"

	"github.com/kdudkov/goatak/cot"
	"github.com/kdudkov/goatak/cotproto"
)

const magicByte = 0xbf

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

		if n < 4 {
			continue
		}

		if buf[0] == magicByte && buf[2] == magicByte {
			if buf[1] == 1 {
				msg := new(cotproto.TakMessage)
				err = proto.Unmarshal(buf[3:n], msg)
				if err != nil {
					app.Logger.Errorf("protobuf decode error: %s", err.Error())
					continue
				}
				xd, err := cot.DetailsFromString(msg.GetCotEvent().GetDetail().GetXmlDetail())
				if err != nil {
					app.Logger.Errorf("protobuf detail extract error: %s", err.Error())
					continue
				}
				scope := msg.GetCotEvent().GetAccess()
				if scope == "" {
					scope = "broadcast"
				}
				app.NewCotMessage(&cot.CotMessage{
					TakMessage: msg,
					Detail:     xd,
					Scope:      scope,
				})
			} else {
				ev := &cot.Event{}
				err = xml.Unmarshal(buf[3:n], ev)
				if err != nil {
					app.Logger.Errorf("xml decode error: %s", err.Error())
					continue
				}

				msg, xd := cot.EventToProto(ev)
				scope := msg.GetCotEvent().GetAccess()
				if scope == "" {
					scope = "broadcast"
				}
				app.NewCotMessage(&cot.CotMessage{
					TakMessage: msg,
					Detail:     xd,
					Scope:      scope,
				})
			}
		} else {
			evt := &cot.Event{}
			if err := xml.Unmarshal(buf[:n], evt); err != nil {
				app.Logger.Errorf("decode error: %v", err)
				continue
			}
			msg, xd := cot.EventToProto(evt)

			app.NewCotMessage(&cot.CotMessage{
				TakMessage: msg,
				Detail:     xd,
				Scope:      "broadcast",
			})
		}
	}

	return nil
}
