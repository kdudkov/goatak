package model

import (
	"time"

	"github.com/kdudkov/goatak/cot"
)

type ChatMessage struct {
	Time    time.Time `json:"time"`
	From    string    `json:"from"`
	To      string    `json:"to"`
	Text    string    `json:"text"`
	FromUid string    `json:"from_Uid"`
	ToUid   string    `json:"to_uid"`
}

// direct
// <__chat parent="RootContactGroup" groupOwner="false" chatroom="Cl1" id="ANDROID-05740daaf44f01" senderCallsign="Kott">
// <chatgrp uid0="ANDROID-dc4a1fb7ad4180be" uid1="ANDROID-05740daaf44f01" id="ANDROID-05740daaf44f01"/></__chat>
// <link uid="ANDROID-dc4a1fb7ad4180be" type="a-f-G-U-C" relation="p-p"/>
// <remarks source="BAO.F.ATAK.ANDROID-dc4a1fb7ad4180be" to="ANDROID-05740daaf44f01" time="2021-04-10T16:40:57.445Z">Roger</remarks>
// <__serverdestination destinations="192.168.0.15:4242:tcp:ANDROID-dc4a1fb7ad4180be"/><marti><dest callsign="Cl1"/></marti>
// chatroom
// <__chat parent="RootContactGroup" groupOwner="false" chatroom="All Chat Rooms" id="All Chat Rooms" senderCallsign="Kott">
// <chatgrp uid0="ANDROID-dc4a1fb7ad4180be" uid1="All Chat Rooms" id="All Chat Rooms"/></__chat>
// <link uid="ANDROID-dc4a1fb7ad4180be" type="a-f-G-U-C" relation="p-p"/>
// <remarks source="BAO.F.ATAK.ANDROID-dc4a1fb7ad4180be" to="All Chat Rooms" time="2021-04-10T16:43:05.294Z">Roger</remarks>
// <__serverdestination destinations="192.168.0.15:4242:tcp:ANDROID-dc4a1fb7ad4180be"/>
// red
// <__chat parent="TeamGroups" groupOwner="false" chatroom="Red" id="Red" senderCallsign="Kott">
// <chatgrp uid0="ANDROID-dc4a1fb7ad4180be" uid1="ANDROID-05740daaf44f01" id="Red"/></__chat>
// <link uid="ANDROID-dc4a1fb7ad4180be" type="a-f-G-U-C" relation="p-p"/>
// <remarks source="BAO.F.ATAK.ANDROID-dc4a1fb7ad4180be" time="2021-04-10T16:44:29.371Z">at VDO</remarks>
// <__serverdestination destinations="192.168.0.15:4242:tcp:ANDROID-dc4a1fb7ad4180be"/><marti><dest callsign="Cl1"/></marti>

func MsgToChat(m *cot.Msg) *ChatMessage {
	chat := m.Detail.GetFirstChild("__chat")
	if chat == nil {
		return nil
	}

	c := &ChatMessage{
		Time: cot.TimeFromMillis(m.TakMessage.CotEvent.StartTime),
		From: chat.GetAttr("senderCallsign"),
		To:   chat.GetAttr("chatroom"),
	}

	if cg := chat.GetFirstChild("chatgrp"); cg != nil {
		c.FromUid = cg.GetAttr("uid0")
		c.ToUid = cg.GetAttr("uid1")
	}
	c.Text, _ = m.Detail.GetChildValue("remarks")

	return c
}
