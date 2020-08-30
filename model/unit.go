package model

import (
	"time"

	"goatac/cot"
)

type Unit struct {
	Uid      string
	Callsign string
	Time     time.Time
	Stale    time.Time
	Type     string
	Icon     string
	evt      *cot.Event
}

func FromEvent(evt *cot.Event) *Unit {

	return &Unit{
		Uid:      evt.Uid,
		Callsign: evt.GetCallsign(),
		Time:     time.Now(),
		Stale:    evt.Stale,
		Type:     evt.Type,
		evt:      evt,
	}
}
