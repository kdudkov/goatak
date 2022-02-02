package cot

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kdudkov/goatak/cotproto"
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
	xd.node.AddChild("link", map[string]string{"uid": uid, "type": typ, "relation": "p-p"})
	msg.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}
	return msg
}

// direct
// <__chat parent="RootContactGroup" groupOwner="false" chatroom="Cl1" id="ANDROID-05740daaf44f01" senderCallsign="Kott"><chatgrp uid0="ANDROID-dc4a1fb7ad4180be" uid1="ANDROID-05740daaf44f01" id="ANDROID-05740daaf44f01"/></__chat>
// <link uid="ANDROID-dc4a1fb7ad4180be" type="a-f-G-U-C" relation="p-p"/><remarks source="BAO.F.ATAK.ANDROID-dc4a1fb7ad4180be" to="ANDROID-05740daaf44f01" time="2021-04-10T16:40:57.445Z">Roger</remarks>
// <__serverdestination destinations="192.168.0.15:4242:tcp:ANDROID-dc4a1fb7ad4180be"/><marti><dest callsign="Cl1"/></marti>
func MakeChatMessage(uid string, callsign string, text string) *cotproto.TakMessage {
	msg := BasicMsg("b-t-f", "server", time.Second*10)
	xd := NewXmlDetails()
	chat := xd.node.AddChild("__chat", map[string]string{"parent": "RootContactGroup", "groupOwner": "false", "chatroom": callsign, "senderCallsign": "Op", "id": uid})
	chat.AddChild("chatgrp", map[string]string{"uid0": "serverop", "uid1": uid, "id": uid})
	xd.node.AddChildWithContext("remarks", nil, text)
	marti := xd.node.AddChild("marti", nil)
	marti.AddChild("dest", map[string]string{"callsign": callsign})
	fmt.Println(xd.AsXMLString())
	msg.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}
	return msg
}
