package model

import "time"

type Unit struct {
	Uid      string
	Callsign string
	Lastseen time.Time
	Type     string
}
