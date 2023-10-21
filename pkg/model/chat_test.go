package model

import (
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestChat(t *testing.T) {
	// user1 (uid1) -> user2 (uid2)

	d := "<__chat parent=\"RootContactGroup\" groupOwner=\"false\" messageId=\"4de0262c-633f-46eb-b8e5-5ef1eb1e5e22\" chatroom=\"user2\" id=\"uid2\" senderCallsign=\"user1\">" +
		"<chatgrp uid0=\"uid1\" uid1=\"uid2\" id=\"uid2\"/></__chat>" +
		"<link uid=\"uid1\" type=\"a-f-G-U-C\" relation=\"p-p\"/>" +
		"<__serverdestination destinations=\"1.1.1.1:4242:tcp:uid1\"/>" +
		"<remarks source=\"BAO.F.ATAK.uid1\" to=\"uid2\" time=\"2023-10-21T20:28:58.991Z\">at breach</remarks>" +
		"<marti><dest callsign=\"user2\"/></marti>"

	m := cot.BasicMsg("b-t-f", "GeoChat.uid1.uid2.4de0262c-633f-46eb-b8e5-5ef1eb1e5e22", time.Minute)
	xd, _ := cot.DetailsFromString(d)
	m.CotEvent.Detail = &cotproto.Detail{XmlDetail: xd.AsXMLString()}
	msg := cot.CotMessage{TakMessage: m, Detail: xd}

	assert.Equal(t, msg.IsChat(), true)

	cm := MsgToChat(&msg)

	assert.Equal(t, cm.From, "user1")
	assert.Equal(t, cm.FromUid, "uid1")
	assert.Equal(t, cm.ToUid, "uid2")
	assert.Equal(t, cm.Chatroom, "user2")
	assert.Equal(t, cm.Direct, true)
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
	msg := cot.CotMessage{TakMessage: m, Detail: xd}

	assert.Equal(t, msg.IsChatReceipt(), true)

	//cm := MsgToChat(&msg)
	//
	//assert.Equal(t, cm.From, "user1")
	//assert.Equal(t, cm.FromUid, "uid1")
	//assert.Equal(t, cm.ToUid, "uid2")
	//assert.Equal(t, cm.Chatroom, "user2")
	//assert.Equal(t, cm.Direct, true)
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
	msg := cot.CotMessage{TakMessage: m, Detail: xd}

	assert.Equal(t, msg.IsChatReceipt(), true)
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
	msg := cot.CotMessage{TakMessage: m, Detail: xd}

	assert.Equal(t, msg.IsChat(), true)

	cm := MsgToChat(&msg)

	assert.Equal(t, cm.From, "user1")
	assert.Equal(t, cm.FromUid, "uid1")
	assert.Equal(t, cm.ToUid, "Red")
	assert.Equal(t, cm.Chatroom, "Red")
	assert.Equal(t, cm.Direct, false)
	assert.Equal(t, cm.Text, "Roger")
}
