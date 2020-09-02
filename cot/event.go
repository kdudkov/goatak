package cot

import (
	"fmt"
	"strings"
	"time"

	"github.com/kdudkov/goatak/xml"
)

type Event struct {
	XMLName xml.Name  `xml:"event"`
	Version string    `xml:"version,attr"`
	Type    string    `xml:"type,attr"`
	Uid     string    `xml:"uid,attr"`
	Time    time.Time `xml:"time,attr"`
	Start   time.Time `xml:"start,attr"`
	Stale   time.Time `xml:"stale,attr"`
	How     string    `xml:"how,attr"`

	Point  Point  `xml:"point,selfclose"`
	Detail Detail `xml:"detail,selfclose"`
}

func (e *Event) String() string {
	if e == nil {
		return "nil"
	}
	return fmt.Sprintf("version=%s, type=%s, uid=%s, how=%s, stale=%s, detail={%s}", e.Version, e.Type, e.Uid, e.How, e.Stale.Sub(e.Start), e.Detail)
}

type Point struct {
	XMLName xml.Name `xml:"point"`
	Lat     float64  `xml:"lat,attr"`
	Lon     float64  `xml:"lon,attr"`
	Hae     float64  `xml:"hae,attr"`
	Ce      float64  `xml:"ce,attr"`
	Le      float64  `xml:"le,attr"`
}

type Detail struct {
	Uid               *Uid               `xml:"uid,omitempty,selfclose"`
	TakVersion        *TakVersion        `xml:"takv,omitempty,selfclose"`
	Contact           *Contact           `xml:"contact,omitempty,selfclose"`
	PrecisionLocation *Precisionlocation `xml:"precisionlocation,omitempty,selfclose"`
	Group             *Group             `xml:"__group,omitempty,selfclose"`
	Status            *Status            `xml:"status,omitempty,selfclose"`
	Usericon          *Usericon          `xml:"usericon,omitempty,selfclose"`
	Track             *Track             `xml:"track,omitempty,selfclose"`
	Chat              *Chat              `xml:"__chat,omitempty"`
	Link              []*Link            `xml:"link,omitempty,selfclose"`
	Remarks           *Remarks           `xml:"remarks,omitempty"`
	Marti             *Marti             `xml:"marti,omitempty"`
	Color             *struct {
		Value string `xml:"argb,attr,omitempty"`
	} `xml:"color,omitempty,selfclose"`
	StrokeColor *struct {
		Value string `xml:"value,attr,omitempty"`
	} `xml:"strokeColor,omitempty,selfclose"`
	FillColor *struct {
		Value string `xml:"value,attr,omitempty"`
	} `xml:"fillColor,omitempty,selfclose"`
	StrokeWeight *struct {
		Value string `xml:"value,attr,omitempty"`
	} `xml:"strokeWeight,omitempty,selfclose"`
}

func (d Detail) String() string {
	var s string

	if d.Uid != nil {
		s += fmt.Sprintf("uid={%s}", d.Uid)
	}
	if d.TakVersion != nil {
		s += fmt.Sprintf(", takv={%s}", d.TakVersion)
	}
	if d.Contact != nil {
		s += fmt.Sprintf(", contact={%s}", d.Contact)
	}
	if d.Group != nil {
		s += fmt.Sprintf(", group={%s}", d.Group)
	}
	if d.Status != nil {
		s += fmt.Sprintf(", status={%s}", d.Status)
	}
	if d.Chat != nil {
		s += fmt.Sprintf(", chat={%s}", d.Chat)
	}
	return strings.TrimLeft(s, " ,")
}

type Contact struct {
	Endpoint string `xml:"endpoint,attr,omitempty"`
	Callsign string `xml:"callsign,attr,omitempty"`
	Phone    string `xml:"phone,attr,omitempty"`
}

func (c *Contact) String() string {
	s := ""
	if c.Endpoint != "" {
		s += fmt.Sprintf(", endpoint=%s", c.Endpoint)
	}
	if c.Callsign != "" {
		s += fmt.Sprintf(", callsign=%s", c.Callsign)
	}
	if c.Phone != "" {
		s += fmt.Sprintf(", phone=%s", c.Phone)
	}
	return strings.TrimLeft(s, ", ")
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
	Name string `xml:"name,attr"`
	Role string `xml:"role,attr"`
}

func (g *Group) String() string {
	if g == nil {
		return "nil"
	}
	return fmt.Sprintf("name=%s, role=%s", g.Name, g.Role)
}

type Status struct {
	Text      string `xml:",chardata"`
	Battery   string `xml:"battery,attr,omitempty"`
	Readiness string `xml:"readiness,attr,omitempty"`
}

type Usericon struct {
	Iconsetpath string `xml:"iconsetpath,attr,omitempty"`
}

type Track struct {
	Course float64 `xml:"course,attr"`
	Speed  float64 `xml:"speed,attr"`
}

type Uid struct {
	Droid string `xml:"Droid,attr,omitempty"`
}

func (u *Uid) String() string {
	if u == nil {
		return "nil"
	}
	return fmt.Sprintf("Droid=%s", u.Droid)
}

type Chat struct {
	Id      string   `xml:"id,attr"`
	Parent  string   `xml:"parent,attr,omitempty"`
	Sender  string   `xml:"senderCallsign,attr,omitempty"`
	Room    string   `xml:"chatroom,attr,omitempty"`
	Owner   string   `xml:"groupOwner,attr,omitempty"`
	ChatGrp *ChatGrp `xml:"chatgrp,omitempty,selfclose"`
}

func (c *Chat) String() string {

	return fmt.Sprintf("id=%s, parent=%s, sender=%s, room=%s, owner=%s, grp={%s}", c.Id, c.Parent, c.Sender, c.Room, c.Owner, c.ChatGrp)
}

type ChatGrp struct {
	Id   string `xml:"id,attr"`
	Uid0 string `xml:"uid0,attr"`
	Uid1 string `xml:"uid1,attr"`
}

func (cg ChatGrp) String() string {
	return fmt.Sprintf("id={%s}, uid0={%s},  uid1={%s}", cg.Id, cg.Uid0, cg.Uid1)
}

type Link struct {
	Time     time.Time `xml:"production_time,attr,omitempty"`
	Relation string    `xml:"relation,attr"`
	Type     string    `xml:"type,attr"`
	Uid      string    `xml:"uid,attr"`
	Point    string    `xml:"point,attr"`
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
	Dest []MartiDest `xml:"dest,omitempty"`
}

type MartiDest struct {
	Callsign string `xml:"callsign,attr,omitempty"`
}

func (e *Event) GetCallsign() string {
	if e.Detail.Contact != nil {
		return e.Detail.Contact.Callsign
	}
	return ""
}

func (e *Event) GetCallsignTo() []string {
	if e.Detail.Marti != nil {
		res := make([]string, len(e.Detail.Marti.Dest))
		for i, d := range e.Detail.Marti.Dest {
			res[i] = d.Callsign
		}
		return res
	}
	return nil
}

func (e *Event) GetDroid() string {
	if e.Detail.Uid != nil {
		return e.Detail.Uid.Droid
	}
	return ""
}

func (e *Event) IsChat() bool {
	return e.Detail.Chat != nil
}

func (e *Event) GetText() string {
	if e.Detail.Remarks != nil {
		return e.Detail.Remarks.Text
	}

	return ""
}

func BasicEvent(typ string, uid string, stale time.Duration) *Event {
	return &Event{
		Version: "2.0",
		Uid:     uid,
		Type:    typ,
		Time:    time.Now().UTC(),
		Start:   time.Now().UTC(),
		Stale:   time.Now().Add(stale).UTC(),
		How:     "m-g",
		Point: Point{
			Lat: 0,
			Lon: 0,
			Hae: 0,
			Ce:  9999999,
			Le:  9999999,
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

func MakePing(uid string) *Event {
	return BasicEvent("t-x-c-t", uid+"-ping", time.Second*10)
}

func MakePong() *Event {
	ev := BasicEvent("t-x-c-t-r", "takPong", time.Second*20)
	ev.How = "h-g-i-g-o"
	return ev
}
