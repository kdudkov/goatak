package main

import (
	"encoding/xml"
	"fmt"
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

type MessageCb func(msg *cot.CotMessage)

type UdpClientHandler struct {
	conn      *net.UDPConn
	addr      *net.UDPAddr
	logger    *slog.Logger
	messageCb MessageCb
	ch        chan []byte
	active    bool
}

func (c *UdpClientHandler) IsActive() bool {
	return c.active
}

func (c *UdpClientHandler) Listen(addrstr string) error {
	c.logger.Info("listening UDP at " + addrstr)

	var err error
	c.addr, err = net.ResolveUDPAddr("udp", addrstr)
	if err != nil {
		c.logger.Error("error", slog.Any("error", err))
		return err
	}

	c.conn, err = net.ListenMulticastUDP("udp", nil, c.addr)
	if err != nil {
		c.logger.Error("error", slog.Any("error", err))
		return err
	}

	c.conn.SetReadBuffer(65535)
	c.active = true

	go c.writer()
	c.reader()

	return nil
}

func (c *UdpClientHandler) SendMsg(msg *cot.CotMessage) error {
	return c.SendCot(msg.GetTakMessage())
}

func (c *UdpClientHandler) SendCot(msg *cotproto.TakMessage) error {
	dat, err := cot.MakeProtoPacket(msg)
	if err != nil {
		return err
	}
	if c.tryAddPacket(dat) {
		return nil
	}

	return fmt.Errorf("client is off")
}

func (c *UdpClientHandler) tryAddPacket(msg []byte) bool {
	select {
	case c.ch <- msg:
	default:
	}
	return true
}

func (c *UdpClientHandler) writer() {
	for b := range c.ch {
		_, _, err := c.conn.WriteMsgUDP(b, []byte{}, c.addr)
		if err != nil {
			c.logger.Error("send error", slog.Any("error", err))
			c.Stop()
			break
		}
	}
}

func (c *UdpClientHandler) reader() {
	defer c.Stop()

	buf := make([]byte, 65535)
	for {
		n, _, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			c.logger.Error("read error", slog.Any("error", err))
			continue
		}

		if n < 4 {
			continue
		}

		if buf[0] == magicByte && buf[2] == magicByte {
			if buf[1] == 1 {
				msg := new(cotproto.TakMessage)

				err = proto.Unmarshal(buf[3:n], msg)
				if err != nil {
					c.logger.Error("protobuf decode error", slog.Any("error", err))

					continue
				}

				cot, err := cot.CotFromProto(msg, "", cot.BroadcastScope)
				if err != nil {
					c.logger.Error("protobuf detail extract error", slog.Any("error", err))
					continue
				}

				c.messageCb(cot)
			} else {
				ev := &cot.Event{}

				err = xml.Unmarshal(buf[3:n], ev)
				if err != nil {
					c.logger.Error("xml decode error", slog.Any("error", err))
					continue
				}

				cot, err := cot.EventToProtoExt(ev, "", scopeBroadcast)
				if err != nil {
					c.logger.Error("error", slog.Any("error", err))
				}

				c.messageCb(cot)
			}
		} else {
			ev := &cot.Event{}
			if err := xml.Unmarshal(buf[:n], ev); err != nil {
				c.logger.Error("decode error", slog.Any("error", err))
				continue
			}

			cot, err := cot.EventToProtoExt(ev, "", scopeBroadcast)
			if err != nil {
				c.logger.Error("error", slog.Any("error", err))
			}

			c.messageCb(cot)
		}
	}
}

func (c *UdpClientHandler) Stop() {
	if !c.active {
		return
	}

	c.active = false
	close(c.ch)
	_ = c.conn.Close()
}

func NewUdpClientHandler(logger *slog.Logger, messageCb MessageCb) *UdpClientHandler {
	return &UdpClientHandler{logger: logger, messageCb: messageCb, ch: make(chan []byte, 10)}
}
