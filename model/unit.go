package model

import (
	"time"

	"github.com/kdudkov/goatak/cot"
)

type Unit struct {
	Uid      string
	Callsign string
	Stale    time.Time
	LastSeen time.Time
	Type     string
	evt      *cot.Event
	client   bool
}

func FromEvent(evt *cot.Event, client bool) *Unit {
	return &Unit{
		Uid:      evt.Uid,
		Callsign: evt.GetCallsign(),
		LastSeen: time.Now(),
		Stale:    evt.Stale,
		Type:     evt.Type,
		client:   client,
		evt:      evt,
	}
}
