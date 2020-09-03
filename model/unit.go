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
	Evt      *cot.Event
}

func FromEvent(evt *cot.Event) *Unit {
	return &Unit{
		Uid:      evt.Uid,
		Callsign: evt.GetCallsign(),
		LastSeen: time.Now(),
		Stale:    evt.Stale,
		Type:     evt.Type,
		Evt:      evt,
	}
}
