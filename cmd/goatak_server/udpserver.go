package main

import (
	"context"
	"encoding/xml"
	"net"

	"google.golang.org/protobuf/proto"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

const magicByte = 0xbf

func (app *App) ListenUDP(ctx context.Context, addr string) error {
	app.Logger.Info("listening UDP at " + addr)

	p, err := net.ListenPacket("udp", addr)
	if err != nil {
		app.Logger.Error("error", "error", err)

		return err
	}

	buf := make([]byte, 65535)

	for ctx.Err() == nil {
		n, _, err := p.ReadFrom(buf)
		if err != nil {
			app.Logger.Error("read error", "error", err)

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
					app.Logger.Error("protobuf decode error", "error", err.Error())

					continue
				}

				scope := msg.GetCotEvent().GetAccess()
				if scope == "" {
					scope = "broadcast"
				}

				c, err := cot.CotFromProto(msg, "", scope)
				if err != nil {
					app.Logger.Error("protobuf detail extract error", "error", err.Error())

					continue
				}

				app.NewCotMessage(c)
			} else {
				ev := &cot.Event{}

				err = xml.Unmarshal(buf[3:n], ev)
				if err != nil {
					app.Logger.Error("xml decode error", "error", err.Error())

					continue
				}

				scope := ev.Access
				if scope == "" {
					scope = "broadcast"
				}

				c, err := cot.EventToProtoExt(ev, "", scope)
				if err != nil {
					app.Logger.Error("error", "error", err.Error())
				}

				app.NewCotMessage(c)
			}
		} else {
			ev := &cot.Event{}
			if err := xml.Unmarshal(buf[:n], ev); err != nil {
				app.Logger.Error("decode error", "error", err)

				continue
			}

			c, err := cot.EventToProtoExt(ev, "", "broadcast")
			if err != nil {
				app.Logger.Error("error", "error", err.Error())
			}

			app.NewCotMessage(c)
		}
	}

	return nil
}
