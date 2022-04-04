package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kdudkov/goatak/cot"
)

type WebUnit struct {
	UID            string    `json:"uid"`
	Callsign       string    `json:"callsign"`
	Category       string    `json:"category"`
	Team           string    `json:"team"`
	Role           string    `json:"role"`
	Time           time.Time `json:"time"`
	LastSeen       time.Time `json:"last_seen"`
	StaleTime      time.Time `json:"stale_time"`
	StartTime      time.Time `json:"start_time"`
	SendTime       time.Time `json:"send_time"`
	Type           string    `json:"type"`
	Lat            float64   `json:"lat"`
	Lon            float64   `json:"lon"`
	Hae            float64   `json:"hae"`
	Speed          float64   `json:"speed"`
	Course         float64   `json:"course"`
	Sidc           string    `json:"sidc"`
	TakVersion     string    `json:"tak_version"`
	Status         string    `json:"status"`
	Text           string    `json:"text"`
	Color          string    `json:"color"`
	ParentCallsign string    `json:"parent_callsign"`
	ParentUid      string    `json:"parent_uid"`
}

type DigitalPointer struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Name string  `json:"name"`
}

func (c *Contact) ToWeb() *WebUnit {
	c.mx.RLock()
	defer c.mx.RUnlock()

	w := c.Item.ToWeb()
	w.Category = "contact"

	if c.online {
		w.Status = "Online"
	} else {
		w.Status = "Offline"
	}

	if v := c.msg.TakMessage.GetCotEvent().GetDetail().GetTakv(); v != nil {
		w.TakVersion = strings.Trim(fmt.Sprintf("%s %s on %s", v.Platform, v.Version, v.Device), " ")
	}

	return w
}

func (u *Unit) ToWeb() *WebUnit {
	w := u.Item.ToWeb()
	w.Category = "unit"
	w.ParentUid = u.parentUID
	w.ParentCallsign = u.parentCallsign
	return w
}

func (p *Point) ToWeb() *WebUnit {
	w := p.Item.ToWeb()
	w.Category = "point"
	w.ParentUid = p.parentUID
	w.ParentCallsign = p.parentCallsign
	return w
}

func (i Item) ToWeb() *WebUnit {
	evt := i.msg.TakMessage.CotEvent

	w := &WebUnit{
		UID:       i.uid,
		Callsign:  i.callsign,
		Time:      cot.TimeFromMillis(evt.SendTime),
		LastSeen:  i.received,
		StaleTime: i.staleTime,
		StartTime: i.startTime,
		SendTime:  i.sendTime,
		Type:      i.type_,
		Lat:       evt.Lat,
		Lon:       evt.Lon,
		Hae:       evt.Hae,
		Speed:     evt.GetDetail().GetTrack().GetSpeed(),
		Course:    evt.GetDetail().GetTrack().GetCourse(),
		Team:      evt.GetDetail().GetGroup().GetName(),
		Role:      evt.GetDetail().GetGroup().GetRole(),
		Sidc:      getSIDC(i.type_),
	}

	w.Text, _ = i.msg.Detail.GetChildValue("remarks")
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
