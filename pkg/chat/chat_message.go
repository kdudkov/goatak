package chat

import (
	"fmt"
	"html"
	"time"

	"github.com/google/uuid"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

type ChatMessage struct {
	msg *cot.CotMessage
	received time.Time
}

func FromCot(c *cot.CotMessage) *ChatMessage {
	return &ChatMessage{
		msg: c,
		received: time.Now(),
	}
}

func (c *ChatMessage) GetMessageID() string {
	if c == nil || c.msg == nil {
		return ""
	}
	
	return c.msg.GetDetail().GetFirst("__chat").GetAttr("messageId")
}

func (c *ChatMessage) GetChatroom() string {
	if c == nil || c.msg == nil {
		return ""
	}
	
	return c.msg.GetDetail().GetFirst("__chat").GetAttr("chatroom")
}

func (c *ChatMessage) GetCallsignFrom() string {
	if c == nil || c.msg == nil {
		return ""
	}
	
	return c.msg.GetDetail().GetFirst("__chat").GetAttr("senderCallsign")
}

func (c *ChatMessage) GetUIDTo() string {
	if c == nil || c.msg == nil {
		return ""
	}
	
	return c.msg.GetDetail().GetFirst("__chat").GetAttr("id")
}

func (c *ChatMessage) GetUIDFrom() string {
	if c == nil || c.msg == nil {
		return ""
	}
	
	if uid := c.msg.GetDetail().GetFirst("__chat").GetFirst("chatgrp").GetAttr("uid0"); uid != "" {
		return uid
	}
	
	if uid := c.msg.GetFirstLink("p-p").GetAttr("uid"); uid != "" {
		return uid	
	}
	
	return ""
}

func (c *ChatMessage) GetText() string {
	if c == nil || c.msg == nil {
		return ""
	}
	
	if rem := c.msg.GetDetail().GetFirst("remarks"); rem != nil {
		return html.UnescapeString(rem.GetText())
	}
	
	return ""
}

// func MsgToChat(m *cot.CotMessage) *ChatMessage {
// 	chat := m.GetDetail().GetFirst("__chat")
// 	if chat == nil {
// 		return nil
// 	}

// 	c := &ChatMessage{
// 		ID:       chat.GetAttr("messageId"),
// 		Time:     m.GetStartTime(),
// 		Parent:   chat.GetAttr("parent"),
// 		Chatroom: chat.GetAttr("chatroom"),
// 		From:     chat.GetAttr("senderCallsign"),
// 		ToUID:    chat.GetAttr("id"),
// 	}

// 	if cg := chat.GetFirst("chatgrp"); cg != nil {
// 		c.FromUID = cg.GetAttr("uid0")
// 	}

// 	if link := m.GetFirstLink("p-p"); link != nil {
// 		if uid := link.GetAttr("uid"); uid != "" {
// 			c.FromUID = uid
// 		}
// 	}

// 	if c.Chatroom != c.ToUID {
// 		c.Direct = true
// 	}

// 	if rem := m.GetDetail().GetFirst("remarks"); rem != nil {
// 		c.Text = html.UnescapeString(rem.GetText())
// 	} else {
// 		return nil
// 	}

// 	return c
// }
// 

func MakeChatMessage(toUID, fromUID, chatroom, from, parent, text string) *cotproto.TakMessage {
	t := time.Now().UTC().Format(time.RFC3339)
	id := uuid.NewString()
	msgUID := fmt.Sprintf("GeoChat.%s.%s.%s", fromUID, toUID, id)
	msg := cot.BasicMsg("b-t-f", msgUID, time.Second*10)
	msg.CotEvent.How = "h-g-i-g-o"
	xd := cot.NewXMLDetails()
	xd.AddPpLink(fromUID, "", "")

	chat := xd.AddOrChangeChild("__chat", map[string]string{"parent": parent, "groupOwner": "false", "chatroom": chatroom, "senderCallsign": from, "id": toUID, "messageId": id})
	chat.AddOrChangeChild("chatgrp", map[string]string{"uid0": fromUID, "uid1": toUID, "id": toUID})

	xd.AddChild("remarks", map[string]string{"source": "BAO.F.ATAK." + fromUID, "to": toUID, "time": t}, html.EscapeString(text))

	if chatroom != toUID {
		marti := xd.AddChild("marti", nil, "")
		marti.AddChild("dest", map[string]string{"callsign": chatroom}, "")
	}

	msg.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}

	return msg
}