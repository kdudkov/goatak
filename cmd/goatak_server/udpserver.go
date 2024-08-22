package main

import (
	"context"
	"encoding/xml"
	"log/slog"
	"net"

	"google.golang.org/protobuf/proto"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

const (
	magicByte      = 0xbf
	scopeBroadcast = "broadcast"
)

func (app *App) ListenUDP(ctx context.Context, addr string) error {
	app.logger.Info("listening UDP at " + addr)

	p, err := net.ListenPacket("udp", addr)
	if err != nil {
		app.logger.Error("error", slog.Any("error", err))

		return err
	}

	buf := make([]byte, 65535)

	for ctx.Err() == nil {
		n, _, err := p.ReadFrom(buf)
		if err != nil {
			app.logger.Error("read error", slog.Any("error", err))

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
					app.logger.Error("protobuf decode error", slog.Any("error", err))

					continue
				}

				c, err := cot.CotFromProto(msg, "", cot.BroadcastScope)
				if err != nil {
					app.logger.Error("protobuf detail extract error", slog.Any("error", err))

					continue
				}

				app.NewCotMessage(c)
			} else {
				ev := &cot.Event{}

				err = xml.Unmarshal(buf[3:n], ev)
				if err != nil {
					app.logger.Error("xml decode error", slog.Any("error", err))

					continue
				}

				c, err := cot.EventToProtoExt(ev, "", scopeBroadcast)
				if err != nil {
					app.logger.Error("error", slog.Any("error", err))
				}

				app.NewCotMessage(c)
			}
		} else {
			ev := &cot.Event{}
			if err := xml.Unmarshal(buf[:n], ev); err != nil {
				app.logger.Error("decode error", slog.Any("error", err))

				continue
			}

			c, err := cot.EventToProtoExt(ev, "", scopeBroadcast)
			if err != nil {
				app.logger.Error("error", slog.Any("error", err))
			}

			app.NewCotMessage(c)
		}
	}

	return nil
}
