package model

import (
	"fmt"
	"strings"
	"time"
)

type WebUnit struct {
	Uid        string    `json:"uid"`
	Callsign   string    `json:"callsign"`
	Team       string    `json:"team"`
	Role       string    `json:"role"`
	Time       time.Time `json:"time"`
	LastSeen   time.Time `json:"last_seen"`
	Stale      time.Time `json:"stale"`
	Type       string    `json:"type"`
	Lat        float64   `json:"lat"`
	Lon        float64   `json:"lon"`
	Hae        float64   `json:"hae"`
	Speed      float64   `json:"speed"`
	Course     float64   `json:"course"`
	Icon       string    `json:"icon"`
	Sidc       string    `json:"sidc"`
	Text       string    `json:"text"`
	TakVersion string    `json:"tak_version"`
	Status     string    `json:"status"`
}

func (c *Contact) ToWeb() *WebUnit {
	w := &WebUnit{
		Uid:      c.Uid,
		Callsign: c.Callsign,
		Time:     c.Evt.Time,
		LastSeen: c.LastSeen,
		Stale:    c.Stale,
		Type:     c.Type,
		Lat:      c.Evt.Point.Lat,
		Lon:      c.Evt.Point.Lon,
		Hae:      c.Evt.Point.Hae,
		Sidc:     getSIDC(c.Type),
	}

	if c.Online {
		w.Status = "Online"
	} else {
		w.Status = "Offline"
	}

	if c.Evt.Detail.Track != nil {
		w.Speed = c.Evt.Detail.Track.Speed
		w.Course = c.Evt.Detail.Track.Course
	}

	if c.Evt.Detail.Remarks != nil {
		w.Text = c.Evt.Detail.Remarks.Text
	}

	if c.Evt.Detail.Group != nil {
		w.Team = c.Evt.Detail.Group.Name
		w.Role = c.Evt.Detail.Group.Role
	}

	if v := c.Evt.Detail.TakVersion; v != nil {
		w.TakVersion = strings.Trim(fmt.Sprintf("%s %s on %s", v.Platform, v.Version, v.Device), " ")
	}
	return w
}

func (u *Unit) ToWeb() *WebUnit {
	w := &WebUnit{
		Uid:      u.Uid,
		Callsign: u.Callsign,
		Time:     u.Evt.Time,
		LastSeen: u.Received,
		Stale:    u.Stale,
		Type:     u.Type,
		Lat:      u.Evt.Point.Lat,
		Lon:      u.Evt.Point.Lon,
		Hae:      u.Evt.Point.Hae,
		Sidc:     getSIDC(u.Type),
	}

	if u.Evt.Detail.Usericon != nil {
		w.Icon = u.Evt.Detail.Usericon.Iconsetpath
	}

	if u.Evt.Detail.Track != nil {
		w.Speed = u.Evt.Detail.Track.Speed
		w.Course = u.Evt.Detail.Track.Course
	}

	if u.Evt.Detail.Remarks != nil {
		w.Text = u.Evt.Detail.Remarks.Text
	}

	if u.Evt.Detail.Group != nil {
		w.Team = u.Evt.Detail.Group.Name
		w.Role = u.Evt.Detail.Group.Role
	}

	return w
}

func getSIDC(fn string) string {
	if !strings.HasPrefix(fn, "a-") {
		return ""
	}

	sidc := "S" + string(fn[2]) + string(fn[4]) + "-"
	if len(fn) > 6 {
		sidc += strings.ReplaceAll(fn[6:], "-", "")
	}

	if len(sidc) < 10 {
		sidc += strings.Repeat("-", 10-len(sidc))
	}
	return strings.ToUpper(sidc)
}
