package cot

import (
	"time"

	"github.com/google/uuid"
	"github.com/kdudkov/goatak/cotproto"
	"github.com/kdudkov/goatak/cotxml"
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
	xd := &cotxml.XMLDetail{Link: []*cotxml.Link{{Uid: uid, Type: typ, Relation: "p-p"}}}

	msg.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.String()}

	return msg
}
