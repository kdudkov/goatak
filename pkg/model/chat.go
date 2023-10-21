package model

import (
	"fmt"
	"github.com/kdudkov/goatak/pkg/cotproto"
	"golang.org/x/net/html"
	"sync"
	"time"

	"github.com/kdudkov/goatak/pkg/cot"
)

type Messages struct {
	mx       sync.RWMutex
	uid      string
	Chats    map[string]*Chat
	Contacts sync.Map
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
	FromUid  string    `json:"from_uid"`
	ToUid    string    `json:"to_uid"`
	Direct   bool      `json:"direct"`
	Text     string    `json:"text"`
}

func NewMessages(myUid string) *Messages {
	return &Messages{
		mx:  sync.RWMutex{},
		uid: myUid,
		Chats: map[string]*Chat{
			"All Chat Rooms": {
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

	uid := msg.FromUid
	callsign := msg.From
	if uid == m.uid {
		uid = msg.ToUid
		callsign = msg.Chatroom
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

func (m *ChatMessage) String() string {
	return fmt.Sprintf("Chat %s (%s) -> %s (%s) \"%s\"", m.From, m.FromUid, m.Chatroom, m.ToUid, m.Text)
}

func MsgToChat(m *cot.CotMessage) *ChatMessage {
	chat := m.Detail.GetFirst("__chat")
	if chat == nil {
		return nil
	}

	c := &ChatMessage{
		Id:       chat.GetAttr("messageId"),
		Time:     cot.TimeFromMillis(m.TakMessage.CotEvent.StartTime),
		Parent:   chat.GetAttr("parent"),
		Chatroom: chat.GetAttr("chatroom"),
		From:     chat.GetAttr("senderCallsign"),
		ToUid:    chat.GetAttr("id"),
	}

	if cg := chat.GetFirst("chatgrp"); cg != nil {
		c.FromUid = cg.GetAttr("uid0")
	}

	if link := m.GetFirstLink("p-p"); link != nil {
		if uid := link.GetAttr("uid"); uid != "" {
			c.FromUid = uid
		}
	}

	if c.Chatroom != c.ToUid {
		c.Direct = true
	}

	if rem := m.Detail.GetFirst("remarks"); rem != nil {
		c.Text = html.UnescapeString(rem.GetText())
	} else {
		return nil
	}

	return c
}

func MakeChatMessage(c *ChatMessage) *cotproto.TakMessage {
	t := time.Now().UTC().Format(time.RFC3339)
	msgUid := fmt.Sprintf("GeoChat.%s.%s.%s", c.FromUid, c.ToUid, c.Id)
	msg := cot.BasicMsg("b-t-f", msgUid, time.Second*10)
	xd := cot.NewXmlDetails()
	xd.AddPpLink(c.FromUid, "", "")

	chat := xd.AddChild("__chat", map[string]string{"parent": c.Parent, "groupOwner": "false", "chatroom": c.Chatroom, "senderCallsign": c.From, "id": c.ToUid, "messageId": c.Id}, "")
	chat.AddChild("chatgrp", map[string]string{"uid0": c.FromUid, "uid1": c.ToUid, "id": c.ToUid}, "")

	xd.AddChild("remarks", map[string]string{"source": "BAO.F.ATAK." + c.FromUid, "to": c.ToUid, "time": t}, html.EscapeString(c.Text))

	if c.Direct {
		marti := xd.AddChild("marti", nil, "")
		marti.AddChild("dest", map[string]string{"callsign": c.Chatroom}, "")
	}

	msg.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}
	return msg
}
