package main

import (
	"encoding/xml"
	"net"

	"google.golang.org/protobuf/proto"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
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

				scope := msg.GetCotEvent().GetAccess()
				if scope == "" {
					scope = "broadcast"
				}

				c, err := cot.CotFromProto(msg, "", scope)
				if err != nil {
					app.Logger.Errorf("protobuf detail extract error: %s", err.Error())

					continue
				}

				app.NewCotMessage(c)
			} else {
				ev := &cot.Event{}

				err = xml.Unmarshal(buf[3:n], ev)
				if err != nil {
					app.Logger.Errorf("xml decode error: %s", err.Error())

					continue
				}

				scope := ev.Access
				if scope == "" {
					scope = "broadcast"
				}

				c, err := cot.EventToProtoExt(ev, "", scope)
				if err != nil {
					app.Logger.Errorf("%s", err.Error())
				}

				app.NewCotMessage(c)
			}
		} else {
			ev := &cot.Event{}
			if err := xml.Unmarshal(buf[:n], ev); err != nil {
				app.Logger.Errorf("decode error: %v", err)

				continue
			}

			c, err := cot.EventToProtoExt(ev, "", "broadcast")
			if err != nil {
				app.Logger.Errorf("%s", err.Error())
			}

			app.NewCotMessage(c)
		}
	}

	return nil
}
