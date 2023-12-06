package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
	"github.com/stretchr/testify/assert"
)

func getChatMsg(msgID, uidFrom, userFrom, uidTo, userTo, text string) *cot.CotMessage {
	d := fmt.Sprintf("<__chat parent=\"RootContactGroup\" groupOwner=\"false\" messageId=\"%[1]s\" chatroom=\"%[5]s\" id=\"%[4]s\" senderCallsign=\"%[3]s\">"+
		"<chatgrp uid0=\"%[2]s\" uid1=\"%[4]s\" id=\"%[4]s\"/></__chat>"+
		"<link uid=\"%[2]s\" type=\"a-f-G-U-C\" relation=\"p-p\"/>"+
		"<__serverdestination destinations=\"1.1.1.1:4242:tcp:%[2]s\"/>"+
		"<remarks source=\"BAO.F.ATAK.%[2]s\" to=\"%[4]s\" time=\"2023-10-21T20:28:58.991Z\">%[6]s</remarks>"+
		"<marti><dest callsign=\"%[5]s\"/></marti>", msgID, uidFrom, userFrom, uidTo, userTo, text)

	m := cot.BasicMsg("b-t-f", fmt.Sprintf("GeoChat.%s.%s.%s", uidFrom, uidTo, msgID), time.Minute)
	xd, _ := cot.DetailsFromString(d)
	m.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()} //nolint:exhaustruct

	return &cot.CotMessage{TakMessage: m, Detail: xd, From: "", Scope: ""}
}

func TestChatFromMe(t *testing.T) {
	msg := getChatMsg("4de0262c-633f-46eb-b8e5-5ef1eb1e5e22", "uid1", "user1", "uid2", "user2", "at breach")
	assert.True(t, msg.IsChat())
	cm := MsgToChat(msg)

	assert.Equal(t, "user1", cm.From)
	assert.Equal(t, "uid1", cm.FromUID)
	assert.Equal(t, "user2", cm.Chatroom)
	assert.Equal(t, "uid2", cm.ToUID)
	assert.True(t, cm.Direct)

	messages := NewMessages("uid1")
	messages.Add(cm)
	ch, ok := messages.Chats["uid2"]
	assert.True(t, ok)
	assert.Equal(t, "uid2", ch.UID)
	assert.Equal(t, "user2", ch.From)
}

func TestChatTomMe(t *testing.T) {
	msg := getChatMsg("4de0262c-633f-46eb-b8e5-5ef1eb1e5e22", "uid1", "user1", "uid2", "user2", "at breach")
	assert.True(t, msg.IsChat())
	cm := MsgToChat(msg)

	assert.Equal(t, "user1", cm.From)
	assert.Equal(t, "uid1", cm.FromUID)
	assert.Equal(t, "user2", cm.Chatroom)
	assert.Equal(t, "uid2", cm.ToUID)
	assert.True(t, cm.Direct)

	messages := NewMessages("uid2")
	messages.Add(cm)
	ch, ok := messages.Chats["uid1"]
	assert.True(t, ok)
	assert.Equal(t, "uid1", ch.UID)
	assert.Equal(t, "user1", ch.From)
}

func TestBtfd(t *testing.T) {
	d := "<__chatreceipt parent=\"RootContactGroup\" groupOwner=\"false\" messageId=\"4de0262c-633f-46eb-b8e5-5ef1eb1e5e22\" chatroom=\"user1\" id=\"uid1\" senderCallsign=\"user2\">" +
		"<chatgrp uid0=\"uid2\" uid1=\"uid1\" id=\"uid1\"/></__chatreceipt>" +
		"<link uid=\"uid2\" type=\"a-f-G-U-C\" relation=\"p-p\"/>" +
		"<__serverdestination destinations=\"2.2.2.2:4242:tcp:uid2\"/>" +
		"<marti><dest callsign=\"user1\"/></marti>"

	m := cot.BasicMsg("b-t-f-d", "4de0262c-633f-46eb-b8e5-5ef1eb1e5e22", time.Minute)
	xd, _ := cot.DetailsFromString(d)
	m.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}
	msg := cot.CotMessage{TakMessage: m, Detail: xd, From: "", Scope: ""}

	assert.True(t, msg.IsChatReceipt())
}

func TestBtfr(t *testing.T) {
	d := "<__chatreceipt parent=\"RootContactGroup\" groupOwner=\"false\" messageId=\"4de0262c-633f-46eb-b8e5-5ef1eb1e5e22\" chatroom=\"user1\" id=\"uid1\" senderCallsign=\"user2\">" +
		"<chatgrp uid0=\"uid2\" uid1=\"uid1\" id=\"uid1\"/></__chatreceipt>" +
		"<link uid=\"uid2\" type=\"a-f-G-U-C\" relation=\"p-p\"/>" +
		"<__serverdestination destinations=\"2.2.2.2:4242:tcp:uid2\"/>" +
		"<marti><dest callsign=\"user1\"/></marti>"

	m := cot.BasicMsg("b-t-f-r", "4de0262c-633f-46eb-b8e5-5ef1eb1e5e22", time.Minute)
	xd, _ := cot.DetailsFromString(d)
	m.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}
	msg := cot.CotMessage{TakMessage: m, Detail: xd, From: "", Scope: ""}

	assert.True(t, msg.IsChatReceipt())
}

func TestMsgRed(t *testing.T) {
	// user1 - Red
	d := "<__chat parent=\"TeamGroups\" groupOwner=\"false\" messageId=\"9f46716f-c875-43b0-8162-3da74196353f\" chatroom=\"Red\" id=\"Red\" senderCallsign=\"user1\">" +
		"<chatgrp uid2=\"uid3\" uid0=\"uid1\" uid1=\"uid2\" id=\"Red\"/></__chat>" +
		"<link uid=\"uid1\" type=\"a-f-G-U-C\" relation=\"p-p\"/>" +
		"<__serverdestination destinations=\"1.1.1.1:4242:tcp:uid1\"/>" +
		"<remarks source=\"BAO.F.ATAK.uid1\" time=\"2023-10-21T21:06:00.852Z\">Roger</remarks>" +
		"<marti><dest callsign=\"user2\"/></marti>"

	m := cot.BasicMsg("b-t-f", "GeoChat.uid1.Red.9f46716f-c875-43b0-8162-3da74196353f", time.Minute)
	xd, _ := cot.DetailsFromString(d)
	m.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}
	msg := cot.CotMessage{TakMessage: m, Detail: xd, From: "", Scope: ""}

	assert.True(t, msg.IsChat())

	cm := MsgToChat(&msg)

	assert.Equal(t, "user1", cm.From)
	assert.Equal(t, "uid1", cm.FromUID)
	assert.Equal(t, "Red", cm.ToUID)
	assert.Equal(t, "Red", cm.Chatroom)
	assert.False(t, cm.Direct)
	assert.Equal(t, "Roger", cm.Text)

	messages := NewMessages("uid2")
	messages.Add(cm)
	ch, ok := messages.Chats["Red"]
	assert.True(t, ok)
	assert.Equal(t, "Red", ch.UID)
	assert.Equal(t, "Red", ch.From)

	messages2 := NewMessages("uid1")
	messages2.Add(cm)
	ch, ok = messages2.Chats["Red"]
	assert.True(t, ok)
	assert.Equal(t, "Red", ch.UID)
	assert.Equal(t, "Red", ch.From)
}
