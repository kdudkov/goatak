package model

import (
	"time"

	"gotac/cot"
)

type Unit struct {
	Uid      string
	Callsign string
	Time     time.Time
	Stale    time.Time
	Type     string
	evt      *cot.Event
}
