package main

import (
	"gotac/cot"
	"gotac/xml"
	"net"
)

func (app *App) ListenUDP(addr string) error {
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

		evt := &cot.Event{}
		if err := xml.Unmarshal(buf[:n], evt); err != nil {
			app.Logger.Errorf("decode error: %v", err)
			continue
		}

		app.ch <- &Msg{event: evt, dat: buf[:n]}
	}

	return nil
}
