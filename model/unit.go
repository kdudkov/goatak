package model

import (
	"time"

	"github.com/kdudkov/goatak/cot"
)

type Unit struct {
	Uid      string
	Callsign string
	Time     time.Time
	Stale    time.Time
	Type     string
	evt      *cot.Event
	client   bool
}

func FromEvent(evt *cot.Event, client bool) *Unit {
	return &Unit{
		Uid:      evt.Uid,
		Callsign: evt.GetCallsign(),
		Time:     time.Now(),
		Stale:    evt.Stale,
		Type:     evt.Type,
		client:   client,
		evt:      evt,
	}
}
