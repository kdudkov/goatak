package model

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/net/html"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

type Messages struct {
	mx       sync.RWMutex
	uid      string
	Chats    map[string]*Chat
	Contacts sync.Map
}

type Chat struct {
	From     string         `json:"from"`
	UID      string         `json:"uid"`
	Messages []*ChatMessage `json:"messages"`
}

type ChatMessage struct {
	ID       string    `json:"message_id"`
	Time     time.Time `json:"time"`
	Parent   string    `json:"parent"`
	Chatroom string    `json:"chatroom"`
	From     string    `json:"from"`
	FromUID  string    `json:"from_uid"`
	ToUID    string    `json:"to_uid"`
	Direct   bool      `json:"direct"`
	Text     string    `json:"text"`
}

func NewMessages(myUID string) *Messages {
	msg := new(Messages)
	msg.uid = myUID
	msg.Chats = map[string]*Chat{
		"All Chat Rooms": {
			From:     "All Chat Rooms",
			UID:      "All Chat Rooms",
			Messages: nil,
		},
	}

	return msg
}

func (m *Messages) Add(msg *ChatMessage) {
	m.mx.Lock()
	defer m.mx.Unlock()

	uid := msg.ToUID
	callsign := msg.Chatroom

	if msg.Direct && msg.ToUID == m.uid {
		uid = msg.FromUID
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
			UID:      uid,
			Messages: []*ChatMessage{msg},
		}
	}
}

func (m *ChatMessage) String() string {
	return fmt.Sprintf("Chat %s (%s) -> %s (%s) \"%s\"", m.From, m.FromUID, m.Chatroom, m.ToUID, m.Text)
}

func MsgToChat(m *cot.CotMessage) *ChatMessage {
	chat := m.Detail.GetFirst("__chat")
	if chat == nil {
		return nil
	}

	c := &ChatMessage{
		ID:       chat.GetAttr("messageId"),
		Time:     cot.TimeFromMillis(m.TakMessage.GetCotEvent().GetStartTime()),
		Parent:   chat.GetAttr("parent"),
		Chatroom: chat.GetAttr("chatroom"),
		From:     chat.GetAttr("senderCallsign"),
		ToUID:    chat.GetAttr("id"),
	}

	if cg := chat.GetFirst("chatgrp"); cg != nil {
		c.FromUID = cg.GetAttr("uid0")
	}

	if link := m.GetFirstLink("p-p"); link != nil {
		if uid := link.GetAttr("uid"); uid != "" {
			c.FromUID = uid
		}
	}

	if c.Chatroom != c.ToUID {
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
	msgUID := fmt.Sprintf("GeoChat.%s.%s.%s", c.FromUID, c.ToUID, c.ID)
	msg := cot.BasicMsg("b-t-f", msgUID, time.Second*10)
	msg.CotEvent.How = "h-g-i-g-o"
	xd := cot.NewXMLDetails()
	xd.AddPpLink(c.FromUID, "", "")

	chat := xd.AddOrChangeChild("__chat", map[string]string{"parent": c.Parent, "groupOwner": "false", "chatroom": c.Chatroom, "senderCallsign": c.From, "id": c.ToUID, "messageId": c.ID})
	chat.AddOrChangeChild("chatgrp", map[string]string{"uid0": c.FromUID, "uid1": c.ToUID, "id": c.ToUID})

	xd.AddChild("remarks", map[string]string{"source": "BAO.F.ATAK." + c.FromUID, "to": c.ToUID, "time": t}, html.EscapeString(c.Text))

	if c.Direct {
		marti := xd.AddChild("marti", nil, "")
		marti.AddChild("dest", map[string]string{"callsign": c.Chatroom}, "")
	}

	msg.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}

	return msg
}
