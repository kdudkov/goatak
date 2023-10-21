package cot

import (
	"time"

	"github.com/google/uuid"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

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
			Ce:        9999999,
			Le:        9999999,
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
	xd := NewXmlDetails()
	xd.AddPpLink(uid, typ, "")
	msg.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}
	return msg
}

func MakeDpMsg(uid string, typ string, name string, lat float64, lon float64) *cotproto.TakMessage {
	msg := BasicMsg("b-m-p-s-p-i", uid+".SPI1", time.Second*20)
	msg.CotEvent.How = "h-e"
	msg.CotEvent.Lat = lat
	msg.CotEvent.Lon = lon
	xd := NewXmlDetails()
	xd.AddPpLink(uid, typ, "")
	msg.CotEvent.Detail = &cotproto.Detail{
		XmlDetail: xd.AsXMLString(),
		Contact:   &cotproto.Contact{Callsign: name},
	}
	return msg
}
