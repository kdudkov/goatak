package model

import (
	"time"

	"github.com/kdudkov/goatak/cot"
)

const (
	staleContactDelete = time.Hour * 24
	receiveUnitDelete  = time.Hour * 24
)

type Unit struct {
	Uid      string
	Type     string
	Callsign string
	Stale    time.Time
	Received time.Time
	Evt      *cot.Event
}

type Contact struct {
	Uid      string
	Type     string
	Callsign string
	Stale    time.Time
	LastSeen time.Time
	Evt      *cot.Event
	Online   bool
}

func (u *Unit) IsOld() bool {
	return u.Received.Add(receiveUnitDelete).Before(time.Now())
}

func (c *Contact) IsOld() bool {
	return (!c.Online) && c.LastSeen.Add(staleContactDelete).Before(time.Now())
}

func (c *Contact) Copy() *Contact {
	return &Contact{
		Uid:      c.Uid,
		Type:     c.Type,
		Callsign: c.Callsign,
		Stale:    c.Stale,
		LastSeen: c.LastSeen,
		Evt:      c.Evt,
		Online:   c.Online,
	}
}

func ContactFromEvent(evt *cot.Event) *Contact {
	return &Contact{
		Uid:      evt.Uid,
		Callsign: evt.GetCallsign(),
		LastSeen: time.Now(),
		Stale:    evt.Stale,
		Type:     evt.Type,
		Evt:      evt,
		Online:   true,
	}
}

func UnitFromEvent(evt *cot.Event) *Unit {
	return &Unit{
		Uid:      evt.Uid,
		Callsign: evt.GetCallsign(),
		Stale:    evt.Stale,
		Type:     evt.Type,
		Evt:      evt,
		Received: time.Now(),
	}
}
