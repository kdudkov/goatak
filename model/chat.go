package model

import (
	"fmt"
	"github.com/kdudkov/goatak/cotproto"
	"golang.org/x/net/html"
	"sync"
	"time"

	"github.com/kdudkov/goatak/cot"
)

type Messages struct {
	mx    sync.RWMutex
	uid   string
	Chats map[string]*Chat
}

type Chat struct {
	From     string         `json:"from"`
	Uid      string         `json:"uid"`
	Messages []*ChatMessage `json:"messages"`
}

type ChatMessage struct {
	Id       string    `json:"message_id"`
	Time     time.Time `json:"time"`
	Parent   string    `json:"parent"`
	Chatroom string    `json:"chatroom"`
	From     string    `json:"from"`
	Text     string    `json:"text"`
	FromUid  string    `json:"from_uid"`
	ToUid    string    `json:"to_uid"`
	Direct   bool      `json:"direct"`
}

func NewMessages(myUid string) *Messages {
	return &Messages{
		mx:  sync.RWMutex{},
		uid: myUid,
		Chats: map[string]*Chat{
			"All Chat Rooms": &Chat{
				From:     "All Chat Rooms",
				Uid:      "All Chat Rooms",
				Messages: nil,
			},
		},
	}
}

func (m *Messages) Add(msg *ChatMessage) {
	m.mx.Lock()
	defer m.mx.Unlock()

	var uid, callsign string
	if msg.FromUid == m.uid {
		uid = msg.ToUid
		callsign = msg.Chatroom
	} else {
		uid = msg.FromUid
		callsign = msg.From
	}

	if c, ok := m.Chats[uid]; ok {
		c.Messages = append([]*ChatMessage{msg}, c.Messages...)
		if c.From == "" && callsign != "" {
			c.From = callsign
		}
	} else {
		m.Chats[uid] = &Chat{
			From:     callsign,
			Uid:      uid,
			Messages: []*ChatMessage{msg},
		}
	}
}

func (m *Messages) Get(f func(map[string]*Chat)) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	f(m.Chats)
}

func (m *Messages) CheckCallsing(uid, callsign string) {
	m.mx.Lock()
	defer m.mx.Unlock()

	for _, v := range m.Chats {
		if v.Uid == uid {
			v.From = callsign
			for _, msg := range v.Messages {
				if msg.FromUid == uid {
					msg.From = callsign
				}
			}
		}
	}
}

func (m *ChatMessage) String() string {
	return fmt.Sprintf("Chat %s (%s) -> %s (%s) \"%s\"", m.From, m.FromUid, m.Chatroom, m.ToUid, m.Text)
}

// direct
// <__chat parent="RootContactGroup" groupOwner="false" chatroom="Cl1" id="{{uid_to}}" senderCallsign="Kott">
// <chatgrp uid0="{{uid_from}}" uid1="{{uid_to}}" id="{{uid_to}}"/></__chat>
// <link uid="{{uid_from}}" type="a-f-G-U-C" relation="p-p"/>
// <remarks source="BAO.F.ATAK.{{uid_from}}" to="{{uid_to}}" time="2021-04-10T16:40:57.445Z">Roger</remarks>
// <__serverdestination destinations="192.168.0.15:4242:tcp:{{uid_from}}"/>
// <marti><dest callsign="Cl1"/></marti>

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
// <__serverdestination destinations="192.168.0.15:4242:tcp:ANDROID-dc4a1fb7ad4180be"/>
// <marti><dest callsign="Cl1"/></marti>

// add contact to group
// <__chat parent="UserGroups" groupOwner="true" messageId="82741635-04dc-413b-9b66-289fde3e22f0" chatroom="j" id="c06c2986-f122-4e85-b213-88498a6fe8bb" senderCallsign="Kott">
// <chatgrp uid0="ANDROID-765a942cbe30d010" uid1="ANDROID-80d62ea2265d8a" id="c06c2986-f122-4e85-b213-88498a6fe8bb"/>
// <hierarchy><group uid="UserGroups" name="Groups">
// <group uid="c06c2986-f122-4e85-b213-88498a6fe8bb" name="j">
// <contact uid="ANDROID-80d62ea2265d8a" name="test1"/>
// <contact uid="ANDROID-765a942cbe30d010" name="Kott"/>
// </group></group></hierarchy></__chat>
// <link uid="ANDROID-765a942cbe30d010" type="a-f-G-U-C" relation="p-p"/>
// <__serverdestination destinations="192.168.1.72:4242:tcp:ANDROID-765a942cbe30d010"/>
// <remarks source="BAO.F.ATAK.ANDROID-765a942cbe30d010" time="2022-04-05T08:26:51.718Z">[UPDATED CONTACTS]</remarks>

func MsgToChat(m *cot.CotMessage) *ChatMessage {
	chat := m.Detail.GetFirst("__chat")
	if chat == nil {
		chat = m.Detail.GetFirst("__chatreceipt")
	}
	if chat == nil {
		return nil
	}

	c := &ChatMessage{
		Id:       chat.GetAttr("messageId"),
		Time:     cot.TimeFromMillis(m.TakMessage.CotEvent.StartTime),
		Parent:   chat.GetAttr("parent"),
		Chatroom: chat.GetAttr("chatroom"),
		From:     chat.GetAttr("senderCallsign"),
	}

	if cg := chat.GetFirst("chatgrp"); cg != nil {
		c.FromUid = cg.GetAttr("uid0")
		c.ToUid = cg.GetAttr("uid1")
	}

	if dest := m.Detail.GetFirst("marti").GetFirst("dest"); dest != nil {
		if dest.GetAttr("callsign") != "" {
			c.Direct = true
		}
	}

	if rem := m.Detail.GetFirst("remarks"); rem != nil {
		c.Text = html.UnescapeString(rem.GetText())
	}

	return c
}
func MakeChatMessage(c *ChatMessage) *cotproto.TakMessage {
	t := time.Now().UTC().Format(time.RFC3339)
	msgUid := fmt.Sprintf("GeoChat.%s.%s.%s", c.FromUid, c.ToUid, c.Id)
	msg := cot.BasicMsg("b-t-f", msgUid, time.Second*10)
	xd := cot.NewXmlDetails()
	xd.AddLink(c.FromUid, "", "")

	chat := xd.AddChild("__chat", map[string]string{"parent": "RootContactGroup", "groupOwner": "false", "chatroom": c.Chatroom, "senderCallsign": c.From, "id": c.ToUid, "messageId": c.Id}, "")
	chat.AddChild("chatgrp", map[string]string{"uid0": c.FromUid, "uid1": c.ToUid, "id": c.ToUid}, "")

	xd.AddChild("remarks", map[string]string{"source": "BAO.F.ATAK." + c.FromUid, "to": c.ToUid, "time": t}, html.EscapeString(c.Text))

	if c.Chatroom != c.ToUid {
		marti := xd.AddChild("marti", nil, "")
		marti.AddChild("dest", map[string]string{"callsign": c.Chatroom}, "")
	}

	msg.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}
	return msg
}
