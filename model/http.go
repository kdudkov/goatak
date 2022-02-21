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
	Category   string    `json:"category"`
	Team       string    `json:"team"`
	Role       string    `json:"role"`
	Time       time.Time `json:"time"`
	LastSeen   time.Time `json:"last_seen"`
	StaleTime  time.Time `json:"stale_time"`
	StartTime  time.Time `json:"start_time"`
	SendTime   time.Time `json:"send_time"`
	Type       string    `json:"type"`
	Lat        float64   `json:"lat"`
	Lon        float64   `json:"lon"`
	Hae        float64   `json:"hae"`
	Speed      float64   `json:"speed"`
	Course     float64   `json:"course"`
	Sidc       string    `json:"sidc"`
	TakVersion string    `json:"tak_version"`
	Status     string    `json:"status"`
	Text       string    `json:"text"`
	Color      string    `json:"color"`
}

type DigitalPointer struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Name string  `json:"name"`
}

func (c *Contact) ToWeb() *WebUnit {
	c.mx.RLock()
	defer c.mx.RUnlock()

	evt := c.msg.TakMessage.CotEvent

	w := &WebUnit{
		Uid:       c.uid,
		Callsign:  c.callsign,
		Category:  "contact",
		Time:      cot.TimeFromMillis(evt.SendTime),
		LastSeen:  c.lastSeen,
		StaleTime: c.staleTime,
		StartTime: c.startTime,
		SendTime:  c.sendTime,
		Type:      c.type_,
		Lat:       evt.Lat,
		Lon:       evt.Lon,
		Hae:       evt.Hae,
		Speed:     evt.GetDetail().GetTrack().GetSpeed(),
		Course:    evt.GetDetail().GetTrack().GetCourse(),
		Team:      evt.GetDetail().GetGroup().GetName(),
		Role:      evt.GetDetail().GetGroup().GetRole(),
		Sidc:      getSIDC(c.type_),
	}

	if c.online {
		w.Status = "Online"
	} else {
		w.Status = "Offline"
	}

	if v := evt.GetDetail().GetTakv(); v != nil {
		w.TakVersion = strings.Trim(fmt.Sprintf("%s %s on %s", v.Platform, v.Version, v.Device), " ")
	}

	w.Text, _ = c.msg.Detail.GetChildValue("remarks")
	return w
}

func (u *Unit) ToWeb() *WebUnit {
	evt := u.msg.TakMessage.CotEvent

	w := &WebUnit{
		Uid:       u.uid,
		Callsign:  u.callsign,
		Category:  "unit",
		Time:      cot.TimeFromMillis(evt.SendTime),
		LastSeen:  u.received,
		StaleTime: u.staleTime,
		StartTime: u.startTime,
		SendTime:  u.sendTime,
		Type:      u.type_,
		Lat:       evt.Lat,
		Lon:       evt.Lon,
		Hae:       evt.Hae,
		Speed:     evt.GetDetail().GetTrack().GetSpeed(),
		Course:    evt.GetDetail().GetTrack().GetCourse(),
		Team:      evt.GetDetail().GetGroup().GetName(),
		Role:      evt.GetDetail().GetGroup().GetRole(),
		Sidc:      getSIDC(u.type_),
	}
	w.Text, _ = u.msg.Detail.GetChildValue("remarks")
	return w
}

func (p *Point) ToWeb() *WebUnit {
	evt := p.msg.TakMessage.GetCotEvent()

	w := &WebUnit{
		Uid:       p.uid,
		Callsign:  p.callsign,
		Category:  "point",
		StaleTime: p.staleTime,
		StartTime: p.startTime,
		SendTime:  p.sendTime,
		Type:      p.type_,
		Lat:       evt.Lat,
		Lon:       evt.Lon,
		Hae:       evt.Hae,
		Speed:     evt.GetDetail().GetTrack().GetSpeed(),
		Course:    evt.GetDetail().GetTrack().GetCourse(),
	}
	w.Text, _ = p.msg.Detail.GetChildValue("remarks")
	return w
}

func getSIDC(fn string) string {
	if !strings.HasPrefix(fn, "a-") {
		return ""
	}

	tokens := strings.Split(fn, "-")

	sidc := "S" + tokens[1] + tokens[2] + "-"
	if len(tokens) > 3 {
		for _, c := range tokens[3:] {
			if len(c) > 1 {
				break
			}
			sidc += c
		}
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
