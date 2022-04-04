package cotxml

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

const (
	nilString = "nil"
	mgString  = "m-g"
)

type Event struct {
	XMLName xml.Name  `xml:"event"`
	Version string    `xml:"version,attr"`
	Type    string    `xml:"type,attr"`
	UID     string    `xml:"uid,attr"`
	Time    time.Time `xml:"time,attr"`
	Start   time.Time `xml:"start,attr"`
	Stale   time.Time `xml:"stale,attr"`
	How     string    `xml:"how,attr"`

	Point  Point  `xml:"point"`
	Detail Detail `xml:"detail"`
}

func (e *Event) String() string {
	if e == nil {
		return nilString
	}
	return fmt.Sprintf("version=%s, type=%s, uid=%s, how=%s, stale=%s, detail={%s}", e.Version, e.Type, e.UID, e.How, e.Stale.Sub(e.Start), e.Detail)
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
	UID               *UID               `xml:"uid,omitempty"`
	TakVersion        *TakVersion        `xml:"takv,omitempty"`
	TakControl        *TakControl        `xml:"TakControl,omitempty"`
	Contact           *Contact           `xml:"contact,omitempty"`
	PrecisionLocation *Precisionlocation `xml:"precisionlocation,omitempty"`
	Group             *Group             `xml:"__group,omitempty"`
	Status            *Status            `xml:"status,omitempty"`
	Usericon          *Usericon          `xml:"usericon,omitempty"`
	Track             *Track             `xml:"track,omitempty"`
	Chat              *Chat              `xml:"__chat,omitempty"`
	Link              []*Link            `xml:"link,omitempty"`
	Remarks           *Remarks           `xml:"remarks,omitempty"`
	Marti             *Marti             `xml:"marti,omitempty"`
	Color             *struct {
		Value string `xml:"argb,attr,omitempty"`
	} `xml:"color,omitempty"`
	StrokeColor *struct {
		Value string `xml:"value,attr,omitempty"`
	} `xml:"strokeColor,omitempty"`
	FillColor *struct {
		Value string `xml:"value,attr,omitempty"`
	} `xml:"fillColor,omitempty"`
	StrokeWeight *struct {
		Value string `xml:"value,attr,omitempty"`
	} `xml:"strokeWeight,omitempty"`
}

func (d Detail) String() string {
	var s string

	if d.UID != nil {
		s += fmt.Sprintf("uid={%s}", d.UID)
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
	if d.Marti != nil {
		s += fmt.Sprintf(", marti={%s}", d.Marti)
	}
	if d.Link != nil {
		s += fmt.Sprintf(", link={%s}", d.Link)
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

type TakControl struct {
	TakProtocolSupport *ProtoVersion `xml:"TakProtocolSupport,omitempty"`
	TakRequest         *ProtoVersion `xml:"TakRequest,omitempty"`
	TakResponce        *TakResponse  `xml:"TakResponse,omitempty"`
}

type ProtoVersion struct {
	Version int8 `xml:"version,attr,omitempty"`
}

type TakResponse struct {
	Status bool `xml:"status,attr"`
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
		return nilString
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
	Course string `xml:"course,attr"`
	Speed  string `xml:"speed,attr"`
}

type UID struct {
	Droid string `xml:"Droid,attr,omitempty"`
}

func (u *UID) String() string {
	if u == nil {
		return nilString
	}
	return fmt.Sprintf("Droid=%s", u.Droid)
}

type Chat struct {
	ID      string   `xml:"id,attr"`
	Parent  string   `xml:"parent,attr,omitempty"`
	Sender  string   `xml:"senderCallsign,attr,omitempty"`
	Room    string   `xml:"chatroom,attr,omitempty"`
	Owner   string   `xml:"groupOwner,attr,omitempty"`
	ChatGrp *ChatGrp `xml:"chatgrp,omitempty"`
}

func (c *Chat) String() string {
	return fmt.Sprintf("id=%s, parent=%s, sender=%s, room=%s, owner=%s, grp={%s}", c.ID, c.Parent, c.Sender, c.Room, c.Owner, c.ChatGrp)
}

type ChatGrp struct {
	ID   string `xml:"id,attr"`
	UID0 string `xml:"uid0,attr"`
	UID1 string `xml:"uid1,attr"`
}

func (cg ChatGrp) String() string {
	return fmt.Sprintf("id={%s}, uid0={%s},  uid1={%s}", cg.ID, cg.UID0, cg.UID1)
}

type Link struct {
	Time           time.Time `xml:"production_time,attr,omitempty"`
	Relation       string    `xml:"relation,attr,omitempty"`
	Type           string    `xml:"type,attr,omitempty"`
	ParentCallsign string    `xml:"parent_callsign,attr,omitempty"`
	UID            string    `xml:"uid,attr,omitempty"`
	Point          string    `xml:"point,attr,omitempty"`
}

func (l Link) String() string {
	return fmt.Sprintf("%s to %s %s", l.Relation, l.UID, l.Type)
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
	if e.Detail.UID != nil {
		return e.Detail.UID.Droid
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

func (e *Event) IsContact() bool {
	return strings.HasPrefix(e.Type, "a-f-") && e.Detail.Contact != nil && e.Detail.Contact.Endpoint != ""
}

func (e *Event) IsTakControlRequest() bool {
	return e.Detail.TakControl != nil && e.Detail.TakControl.TakRequest != nil
}

func BasicMsg(typ string, uid string, stale time.Duration) *Event {
	return &Event{
		Version: "2.0",
		UID:     uid,
		Type:    typ,
		Time:    time.Now().UTC(),
		Start:   time.Now().UTC(),
		Stale:   time.Now().Add(stale).UTC(),
		Point: Point{
			Lat: 0,
			Lon: 0,
			Hae: 0,
			Ce:  9999999,
			Le:  9999999,
		},
	}
}

func VersionSupportMsg(ver int8) *Event {
	ev := BasicMsg("t-x-takp-v", "protouid", time.Minute)
	ev.How = mgString
	ev.Detail = Detail{TakControl: &TakControl{TakProtocolSupport: &ProtoVersion{Version: ver}}}
	return ev
}

func VersionReqMsg(ver int8) *Event {
	ev := BasicMsg("t-x-takp-v", "protouid", time.Minute)
	ev.How = mgString
	ev.Detail = Detail{TakControl: &TakControl{TakRequest: &ProtoVersion{Version: ver}}}
	return ev
}

func ProtoChangeOkMsg() *Event {
	ev := BasicMsg("t-x-takp-r", "protouid", time.Minute)
	ev.How = mgString
	ev.Detail = Detail{TakControl: &TakControl{TakResponce: &TakResponse{true}}}
	return ev
}
