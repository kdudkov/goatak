package cot

import (
	"time"

	"github.com/google/uuid"
	"github.com/kdudkov/goatak/pkg/cotproto"
	"google.golang.org/protobuf/proto"
)

const NotNum = 9999999

func BasicMsg(typ string, uid string, stale time.Duration) *cotproto.TakMessage {
	return &cotproto.TakMessage{
		CotEvent: &cotproto.CotEvent{
			Type:      typ,
			Access:    "",
			Qos:       "",
			Opex:      "",
			Uid:       uid,
			SendTime:  TimeToMillis(time.Now()),
			StartTime: TimeToMillis(time.Now()),
			StaleTime: TimeToMillis(time.Now().Add(stale)),
			How:       "m-g",
			Lat:       0,
			Lon:       0,
			Hae:       0,
			Ce:        NotNum,
			Le:        NotNum,
			Detail:    nil,
		},
	}
}

func MakePing(uid string) *cotproto.TakMessage {
	return BasicMsg("t-x-c-t", uid+"-ping", time.Second*10)
}

func MakePong() *cotproto.TakMessage {
	msg := BasicMsg("t-x-c-t-r", "takPong", time.Second*20)
	msg.CotEvent.How = "h-g-i-g-o"
	return msg
}

func MakeOfflineMsg(uid string, typ string) *cotproto.TakMessage {
	msg := BasicMsg("t-x-d-d", uuid.New().String(), time.Minute*3)
	msg.CotEvent.How = "h-g-i-g-o"
	xd := NewXMLDetails()
	xd.AddPpLink(uid, typ, "")
	msg.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}
	return msg
}

func MakeDpMsg(uid string, typ string, name string, lat float64, lon float64) *cotproto.TakMessage {
	msg := BasicMsg("b-m-p-s-p-i", uid+".SPI1", time.Second*20)
	msg.CotEvent.How = "h-e"
	msg.CotEvent.Lat = lat
	msg.CotEvent.Lon = lon
	xd := NewXMLDetails()
	xd.AddPpLink(uid, typ, "")
	msg.CotEvent.Detail = &cotproto.Detail{
		XmlDetail: xd.AsXMLString(),
		Contact:   &cotproto.Contact{Callsign: name},
	}
	return msg
}

func CloneMessageNoCoords(msg *cotproto.TakMessage) *cotproto.TakMessage {
	if msg == nil {
		return nil
	}

	data, _ := proto.Marshal(msg)
	msg1 := new(cotproto.TakMessage)
	_ = proto.Unmarshal(data, msg1)

	if evt := msg1.GetCotEvent(); evt != nil {
		evt.Lat = 0
		evt.Lon = 0
		evt.Hae = 0
		evt.Ce = NotNum
		evt.Le = NotNum
	}

	return msg1
}
