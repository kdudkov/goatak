package model

import (
	"time"

	"github.com/kdudkov/goatak/cot"
)

type ChatMessage struct {
	Time    time.Time `json:"time"`
	Parent  string    `json:"parent"`
	Room    string    `json:"chatroom"`
	From    string    `json:"from"`
	Text    string    `json:"text"`
	FromUid string    `json:"from_Uid"`
	ToUid   string    `json:"to_uid"`
}

// direct
// <__chat parent="RootContactGroup" groupOwner="false" chatroom="Cl1" id="{{uid_to}}" senderCallsign="Kott">
// <chatgrp uid0="{{uid_from}}" uid1="{{uid_to}}" id="{{uid_to}}"/></__chat>
// <link uid="{{uid_from}}" type="a-f-G-U-C" relation="p-p"/>
// <remarks source="BAO.F.ATAK.{{uid_from}}" to="{{uid_to}}" time="2021-04-10T16:40:57.445Z">Roger</remarks>
// <__serverdestination destinations="192.168.0.15:4242:tcp:{{uid_from}}"/><marti><dest callsign="Cl1"/></marti>

// chatroom
// <__chat parent="RootContactGroup" groupOwner="false" chatroom="All Chat Rooms" id="All Chat Rooms" senderCallsign="Kott">
// <chatgrp uid0="{{uid_from}}" uid1="All Chat Rooms" id="All Chat Rooms"/></__chat>
// <link uid="{{uid_from}}" type="a-f-G-U-C" relation="p-p"/>
// <remarks source="BAO.F.ATAK.{{uid_from}}" to="All Chat Rooms" time="2021-04-10T16:43:05.294Z">Roger</remarks>
// <__serverdestination destinations="192.168.0.15:4242:tcp:{{uid_from}}"/>

// red
// <__chat parent="TeamGroups" groupOwner="false" chatroom="Red" id="Red" senderCallsign="Kott">
// <chatgrp uid0="ANDROID-dc4a1fb7ad4180be" uid1="ANDROID-05740daaf44f01" id="Red"/></__chat>
// <link uid="ANDROID-dc4a1fb7ad4180be" type="a-f-G-U-C" relation="p-p"/>
// <remarks source="BAO.F.ATAK.ANDROID-dc4a1fb7ad4180be" time="2021-04-10T16:44:29.371Z">at VDO</remarks>
// <__serverdestination destinations="192.168.0.15:4242:tcp:ANDROID-dc4a1fb7ad4180be"/><marti><dest callsign="Cl1"/></marti>

// add contact to group
// <__chat parent="UserGroups" groupOwner="true" messageId="82741635-04dc-413b-9b66-289fde3e22f0" chatroom="j" id="c06c2986-f122-4e85-b213-88498a6fe8bb" senderCallsign="Kott">
// <chatgrp uid0="ANDROID-765a942cbe30d010" uid1="ANDROID-80d62ea2265d8a" id="c06c2986-f122-4e85-b213-88498a6fe8bb"/>
// <hierarchy><group uid="UserGroups" name="Groups">
//<group uid="c06c2986-f122-4e85-b213-88498a6fe8bb" name="j">
// <contact uid="ANDROID-80d62ea2265d8a" name="test1"/>
// <contact uid="ANDROID-765a942cbe30d010" name="Kott"/>
// </group></group></hierarchy></__chat>
// <link uid="ANDROID-765a942cbe30d010" type="a-f-G-U-C" relation="p-p"/>
// <__serverdestination destinations="192.168.1.72:4242:tcp:ANDROID-765a942cbe30d010"/>
// <remarks source="BAO.F.ATAK.ANDROID-765a942cbe30d010" time="2022-04-05T08:26:51.718Z">[UPDATED CONTACTS]</remarks>

func MsgToChat(m *cot.CotMessage) *ChatMessage {
	chat := m.Detail.GetFirst("__chat")
	if chat == nil {
		return nil
	}

	c := &ChatMessage{
		Time:   cot.TimeFromMillis(m.TakMessage.CotEvent.StartTime),
		Parent: chat.GetAttr("parent"),
		Room:   chat.GetAttr("chatroom"),
		From:   chat.GetAttr("senderCallsign"),
	}

	if cg := chat.GetFirst("chatgrp"); cg != nil {
		c.FromUid = cg.GetAttr("uid0")
		c.ToUid = cg.GetAttr("id")
	}

	if c.From == "" {
		c.From = c.FromUid
	}
	c.Text = m.Detail.GetFirst("remarks").GetText()

	return c
}
