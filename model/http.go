package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kdudkov/goatak/cot"
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
	Sidc       string    `json:"sidc"`
	TakVersion string    `json:"tak_version"`
	Status     string    `json:"status"`
}

func (c *Contact) ToWeb() *WebUnit {
	c.mx.RLock()
	defer c.mx.RUnlock()

	w := &WebUnit{
		Uid:      c.uid,
		Callsign: c.callsign,
		Time:     cot.TimeFromMillis(c.msg.CotEvent.SendTime),
		LastSeen: c.lastSeen,
		Stale:    c.stale,
		Type:     c.type_,
		Lat:      c.msg.CotEvent.Lat,
		Lon:      c.msg.CotEvent.Lon,
		Hae:      c.msg.CotEvent.Hae,
		Sidc:     getSIDC(c.type_),
	}

	if c.online {
		w.Status = "Online"
	} else {
		w.Status = "Offline"
	}

	w.Speed = c.msg.GetCotEvent().GetDetail().GetTrack().GetSpeed()
	w.Course = c.msg.GetCotEvent().GetDetail().GetTrack().GetCourse()
	w.Team = c.msg.GetCotEvent().GetDetail().GetGroup().GetName()
	w.Role = c.msg.GetCotEvent().GetDetail().GetGroup().GetRole()

	if v := c.msg.GetCotEvent().GetDetail().GetTakv(); v != nil {
		w.TakVersion = strings.Trim(fmt.Sprintf("%s %s on %s", v.Platform, v.Version, v.Device), " ")
	}
	return w
}

func (u *Unit) ToWeb() *WebUnit {
	w := &WebUnit{
		Uid:      u.uid,
		Callsign: u.callsign,
		Time:     cot.TimeFromMillis(u.msg.CotEvent.SendTime),
		LastSeen: u.received,
		Stale:    u.stale,
		Type:     u.type_,
		Lat:      u.msg.CotEvent.Lat,
		Lon:      u.msg.CotEvent.Lon,
		Hae:      u.msg.CotEvent.Hae,
		Sidc:     getSIDC(u.type_),
	}

	w.Speed = u.msg.GetCotEvent().GetDetail().GetTrack().GetSpeed()
	w.Course = u.msg.GetCotEvent().GetDetail().GetTrack().GetCourse()
	w.Team = u.msg.GetCotEvent().GetDetail().GetGroup().GetName()
	w.Role = u.msg.GetCotEvent().GetDetail().GetGroup().GetRole()

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

func argb2hex(argb string) string {
	if s, err := strconv.Atoi(argb); err == nil {
		return "#" + fmt.Sprintf("%x", uint32(s))[2:]
	}

	return ""
}
