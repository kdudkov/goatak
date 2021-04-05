package cot

import (
	"time"

	"github.com/google/uuid"
	v0 "github.com/kdudkov/goatak/cot/v0"
	v1 "github.com/kdudkov/goatak/cot/v1"
)

func BasicMsg(typ string, uid string, stale time.Duration) *v1.TakMessage {
	return &v1.TakMessage{
		CotEvent: &v1.CotEvent{
			Type:      typ,
			Access:    "",
			Qos:       "",
			Opex:      "",
			Uid:       uid,
			SendTime:  v1.TimeToMillis(time.Now()),
			StartTime: v1.TimeToMillis(time.Now()),
			StaleTime: v1.TimeToMillis(time.Now().Add(stale)),
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

func MakePing(uid string) *v1.TakMessage {
	return BasicMsg("t-x-c-t", uid+"-ping", time.Second*10)
}

func MakePong() *v1.TakMessage {
	msg := BasicMsg("t-x-c-t-r", "takPong", time.Second*20)
	msg.CotEvent.How = "h-g-i-g-o"
	return msg
}

func MakeOfflineMsg(uid string, typ string) *v1.TakMessage {
	msg := BasicMsg("t-x-d-d", uuid.New().String(), time.Minute*3)
	msg.CotEvent.How = "h-g-i-g-o"
	xd := &v1.XMLDetail{Link: []*v0.Link{{Uid: uid, Type: typ, Relation: "p-p"}}}

	msg.CotEvent.Detail = &v1.Detail{XmlDetail: xd.String()}

	return msg
}
