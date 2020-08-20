package cot

import (
	"encoding/xml"
	"fmt"
	"time"
)

type Point struct {
	XMLName xml.Name `xml:"point"`
	Lat     float64  `xml:"lat,attr"`
	Lon     float64  `xml:"lon,attr"`
	Hae     string   `xml:"hae,attr"`
	Ce      string   `xml:"ce,attr"`
	Le      string   `xml:"le,attr"`
}

type Contact struct {
	Endpoint string `xml:"endpoint,attr,omitempty"`
	Callsign string `xml:"callsign,attr,omitempty"`
	Phone    string `xml:"phone,attr,omitempty"`
}

type TakVersion struct {
	Os       string `xml:"os,attr,omitempty"`
	Version  string `xml:"version,attr,omitempty"`
	Device   string `xml:"device,attr,omitempty"`
	Platform string `xml:"platform,attr,omitempty"`
}

type Precisionlocation struct {
	Altsrc      string `xml:"altsrc,attr"`
	Geopointsrc string `xml:"geopointsrc,attr"`
}

type Group struct {
	Role string `xml:"role,attr"`
	Name string `xml:"name,attr"`
}

type Status struct {
	Text      string `xml:",chardata"`
	Battery   string `xml:"battery,attr,omitempty"`
	Readiness string `xml:"readiness,attr,omitempty"`
}

type Track struct {
	Course float64 `xml:"course,attr"`
	Speed  float64 `xml:"speed,attr"`
}

type Uid struct {
	Droid string `xml:"Droid,attr,omitempty"`
}

type Chat struct {
	Id      string   `xml:"id,attr"`
	Parent  string   `xml:"parent,attr,omitempty"`
	Sender  string   `xml:"senderCallsign,attr,omitempty"`
	Room    string   `xml:"chatroom,attr,omitempty"`
	Owner   string   `xml:"groupOwner,attr,omitempty"`
	ChatGrp *ChatGrp `xml:"chatgrp,omitempty"`
}

func (c Chat) String() string {
	return fmt.Sprintf("id: %s, parent: %s, sender: %s, room: %s, owner: %s, grp: {%s}", c.Id, c.Parent, c.Sender, c.Room, c.Owner, c.ChatGrp)
}

type ChatGrp struct {
	Id   string `xml:"id,attr"`
	Uid0 string `xml:"uid0,attr"`
	Uid1 string `xml:"uid1,attr"`
}

func (cg ChatGrp) String() string {
	return fmt.Sprintf("id: {%s}, uid0: {%s},  uid1: {%s}", cg.Id, cg.Uid0, cg.Uid1)
}

type Link struct {
	Time     time.Time `xml:"production_time,attr,omitempty"`
	Relation string    `xml:"relation,attr"`
	Type     string    `xml:"type,attr"`
	Uid      string    `xml:"uid,attr"`
}

func (l Link) String() string {
	return fmt.Sprintf("%s to %s %s", l.Relation, l.Uid, l.Type)
}

type Remarks struct {
	Time   time.Time `xml:"time,attr,omitempty"`
	To     string    `xml:"to,attr,omitempty"`
	Source string    `xml:"source,attr,omitempty"`
	Text   string    `xml:",chardata"`
}

func (r Remarks) String() string {
	return fmt.Sprintf("to: %s text: %s", r.To, r.Text)
}

type Marti struct {
	Dest *MartiDest `xml:"dest,omitempty"`
}

type MartiDest struct {
	Callsign string `xml:"callsign,attr,omitempty"`
}

type Detail struct {
	Text              string             `xml:",chardata"`
	Uid               *Uid               `xml:"uid"`
	TakVersion        *TakVersion        `xml:"takv,omitempty"`
	Contact           *Contact           `xml:"contact,omitempty"`
	PrecisionLocation *Precisionlocation `xml:"precisionlocation,omitempty"`
	Group             *Group             `xml:"__group,omitempty"`
	Status            *Status            `xml:"status,omitempty"`
	Track             *Track             `xml:"track,omitempty"`
	Chat              *Chat              `xml:"__chat,omitempty"`
	Link              *Link              `xml:"link,omitempty"`
	Remarks           *Remarks           `xml:"remarks,omitempty"`
	Marti             *Marti             `xml:"marti,omitempty"`
}

type Event struct {
	XMLName xml.Name  `xml:"event"`
	Text    string    `xml:",chardata"`
	Version string    `xml:"version,attr"`
	Uid     string    `xml:"uid,attr"`
	Type    string    `xml:"type,attr"`
	Time    time.Time `xml:"time,attr"`
	Start   time.Time `xml:"start,attr"`
	Stale   time.Time `xml:"stale,attr"`
	How     string    `xml:"how,attr"`
	Point   *Point    `xml:"point,omitempty"`
	Detail  *Detail   `xml:"detail,omitempty"`
}

func (e *Event) GetCallsign() string {
	if e.Detail != nil && e.Detail.Contact != nil {
		return e.Detail.Contact.Callsign
	}
	return ""
}

func (e *Event) GetCallsignTo() string {
	if e.Detail != nil && e.Detail.Marti != nil && e.Detail.Marti.Dest != nil {
		return  e.Detail.Marti.Dest.Callsign
	}
	return ""
}

func (e *Event) GetDroid() string {
	if e.Detail != nil && e.Detail.Uid != nil {
		return  e.Detail.Uid.Droid
	}
	return ""
}

func (e *Event) IsChat() bool {
	return e.Detail != nil && e.Detail.Chat != nil
}

func (e *Event) GetText() string {
	if e.Detail != nil && e.Detail.Remarks != nil {
		return e.Detail.Remarks.Text
	}

	return ""
}

func BasicEvent(typ string, uid string) *Event {
	return &Event{
		Version: "2.0",
		Uid:     uid,
		Type:    typ,
		Time:    time.Now().UTC(),
		Start:   time.Now().UTC(),
		Stale:   time.Now().Add(time.Hour).UTC(),
		Point: &Point{
			XMLName: xml.Name{},
			Lat:     0,
			Lon:     0,
			Hae:     "0.0",
			Ce:      "0.0",
			Le:      "999999",
		},
	}
}

func BasicDetail(callsign string, team string, role string) *Detail {
	return &Detail{
		Uid: &Uid{Droid: callsign},
		TakVersion: &TakVersion{
			Os:       "no",
			Version:  "0.1",
			Device:   "Cray 2",
			Platform: "GO-ATAC",
		},
		Contact: &Contact{
			Endpoint: "*:-1:tcp",
			Callsign: callsign,
			Phone:    "",
		},
		PrecisionLocation: nil,
		Group: &Group{
			Role: role,
			Name: team,
		},
		Status: nil,
		Track: &Track{
			Course: 0,
			Speed:  0,
		},
	}
}

func MakeMe(uid string, callsign string) *Event {
	ev := BasicEvent("a-f-G-U-U-S-O", uid)
	ev.Detail = BasicDetail(callsign, "Red", "HQ")

	return ev
}

func MakePos(uid string, callsign string) string {
	tpl := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<event version="2.0" uid="%s" type="a-f-G-U-C" time="%s" start="%s" stale="%s" how="h-e"><point lat="60.2" lon="32.3" hae="65.1" ce="10.7" le="9999999.0"/><detail><takv os="29" version="4.0.0.7 (7939f102).1592931989-CIV" device="XIAOMI MI 9T" platform="ATAK-CIV"/><contact endpoint="*:-1:stcp" callsign="%s"/><uid Droid="%s"/><precisionlocation altsrc="GPS" geopointsrc="GPS"/><__group role="Team Member" name="Dark Green"/><status battery="48"/><track course="213.27765249411488" speed="0.0"/></detail></event>`

	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	stale := time.Now().Add(time.Hour).UTC().Format("2006-01-02T15:04:05Z")

	return fmt.Sprintf(tpl, uid, now, now, stale, callsign, callsign)
}

func MakePing(uid string) string {
	tpl := `<?xml version="1.0"?>
<event version="2.0" uid="%s-ping" type="t-x-c-t" time="%s" start="%s" stale="%s" how="m-g"><point lat="0.00000000" lon="0.00000000" hae="0.00000000" ce="9999999" le="9999999"/><detail/></event>`

	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	stale := time.Now().Add(time.Hour).UTC().Format("2006-01-02T15:04:05Z")

	return fmt.Sprintf(tpl, uid, now, now, stale)
}
