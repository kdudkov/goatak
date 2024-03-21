package main

import (
	"net"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/kdudkov/goatak/pkg/cot"
)

const magicByte = 0xbf

type BroadcastDumper struct {
	n    int
	prev *cot.CotMessage
	conn *net.UDPConn
}

func NewBroadcastDumper(n int) *BroadcastDumper {
	return &BroadcastDumper{
		n: n,
	}
}

func (d *BroadcastDumper) Start() {
	addr, err := net.ResolveUDPAddr("udp", "239.2.3.1:6969")

	if err != nil {
		panic(err)
	}

	d.conn, err = net.DialUDP("udp", nil, addr)

	if err != nil {
		panic(err)
	}
}

func (d *BroadcastDumper) Stop() {
}

func (d *BroadcastDumper) Process(msg *cot.CotMessage) error {
	if d.prev == nil {
		d.prev = msg
		return nil
	}

	if !msg.IsMapItem() {
		return nil
	}

	evt := d.prev.GetTakMessage().GetCotEvent()

	if evt == nil {
		d.prev = msg
		return nil
	}

	t := msg.GetStartTime().Sub(d.prev.GetStartTime())
	stale := msg.GetStartTime().Sub(msg.GetStaleTime())

	m := d.prev
	now := time.Now()
	evt.StartTime = cot.TimeToMillis(now)
	evt.SendTime = cot.TimeToMillis(now)
	evt.StaleTime = cot.TimeToMillis(now.Add(stale / time.Duration(d.n)))

	b, _ := proto.Marshal(m.TakMessage)
	d.conn.Write([]byte{magicByte, 1, magicByte})

	bb := make([]byte, len(b)+3)
	bb[0] = magicByte
	bb[1] = 1
	bb[2] = magicByte
	copy(bb[3:], b)

	d.conn.Write(bb)

	d.prev = msg

	if t > 0 {
		time.Sleep(t / time.Duration(d.n))
	}

	return nil
}
