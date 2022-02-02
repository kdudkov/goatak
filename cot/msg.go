package cot

import (
	"fmt"
	"strings"
	"time"

	"github.com/kdudkov/goatak/cotproto"
)

type Msg struct {
	TakMessage *cotproto.TakMessage
	Detail     *XMLDetails
}

func (m *Msg) GetUid() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	return m.TakMessage.GetCotEvent().Uid
}

func (m *Msg) GetType() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	return m.TakMessage.GetCotEvent().Type
}

func (m *Msg) GetCallsign() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	return m.TakMessage.GetCotEvent().GetDetail().GetContact().GetCallsign()
}

func (m *Msg) GetCallsignTo() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	return m.TakMessage.GetCotEvent().GetDetail().GetContact().GetCallsign()
}

func (m *Msg) GetStale() time.Time {
	if m == nil || m.TakMessage == nil {
		return time.Unix(0, 0)
	}

	return TimeFromMillis(m.TakMessage.CotEvent.StaleTime)
}

func (m *Msg) IsContact() bool {
	if m == nil || m.TakMessage == nil {
		return false
	}

	return strings.HasPrefix(m.GetType(), "a-f-") && m.TakMessage.GetCotEvent().GetDetail().GetContact().GetEndpoint() != ""
}

func (m *Msg) IsChat() bool {
	if m == nil || m.TakMessage == nil {
		return false
	}

	return m.GetType() == "b-t-f" && m.Detail != nil && m.Detail.HasChild("__chat")
}

func (m *Msg) PrintChat() string {
	chat := m.Detail.GetFirstChild("__chat")
	if chat == nil {
		return ""
	}

	from := chat.GetAttr("senderCallsign")
	to := chat.GetAttr("chatroom")
	text, _ := m.Detail.getChildValue("remarks")

	return fmt.Sprintf("%s -> %s: \"%s\"", from, to, text)
}

func TimeFromMillis(ms uint64) time.Time {
	return time.Unix(0, 1000000*int64(ms))
}

func TimeToMillis(t time.Time) uint64 {
	return uint64(t.UnixNano() / 1000000)
}

// direct
// <__chat parent="RootContactGroup" groupOwner="false" chatroom="Cl1" id="ANDROID-05740daaf44f01" senderCallsign="Kott"> <chatgrp uid0="ANDROID-dc4a1fb7ad4180be" uid1="ANDROID-05740daaf44f01" id="ANDROID-05740daaf44f01"/></__chat>
// <link uid="ANDROID-dc4a1fb7ad4180be" type="a-f-G-U-C" relation="p-p"/><remarks source="BAO.F.ATAK.ANDROID-dc4a1fb7ad4180be" to="ANDROID-05740daaf44f01" time="2021-04-10T16:40:57.445Z">Roger</remarks>
// <__serverdestination destinations="192.168.0.15:4242:tcp:ANDROID-dc4a1fb7ad4180be"/><marti><dest callsign="Cl1"/></marti>
// chatroom
// <__chat parent="RootContactGroup" groupOwner="false" chatroom="All Chat Rooms" id="All Chat Rooms" senderCallsign="Kott"><chatgrp uid0="ANDROID-dc4a1fb7ad4180be" uid1="All Chat Rooms" id="All Chat Rooms"/></__chat>
// <link uid="ANDROID-dc4a1fb7ad4180be" type="a-f-G-U-C" relation="p-p"/><remarks source="BAO.F.ATAK.ANDROID-dc4a1fb7ad4180be" to="All Chat Rooms" time="2021-04-10T16:43:05.294Z">Roger</remarks>
// <__serverdestination destinations="192.168.0.15:4242:tcp:ANDROID-dc4a1fb7ad4180be"/>
// red
// <__chat parent="TeamGroups" groupOwner="false" chatroom="Red" id="Red" senderCallsign="Kott"><chatgrp uid0="ANDROID-dc4a1fb7ad4180be" uid1="ANDROID-05740daaf44f01" id="Red"/></__chat>
// <link uid="ANDROID-dc4a1fb7ad4180be" type="a-f-G-U-C" relation="p-p"/><remarks source="BAO.F.ATAK.ANDROID-dc4a1fb7ad4180be" time="2021-04-10T16:44:29.371Z">at VDO</remarks>
// <__serverdestination destinations="192.168.0.15:4242:tcp:ANDROID-dc4a1fb7ad4180be"/><marti><dest callsign="Cl1"/></marti>
