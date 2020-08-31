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
	Stale      time.Time `json:"stale"`
	Type       string    `json:"type"`
	Lat        float64   `json:"lat"`
	Lon        float64   `json:"lon"`
	Hae        float64   `json:"hae"`
	Speed      float64   `json:"speed"`
	Icon       string    `json:"icon"`
	Text       string    `json:"text"`
	TakVersion string    `json:"tak_version"`
}

func (u *Unit) ToWeb() *WebUnit {

	w := &WebUnit{
		Uid:      u.Uid,
		Callsign: u.Callsign,
		Time:     u.Time,
		Stale:    u.Stale,
		Type:     u.Type,
		Lat:      u.evt.Point.Lat,
		Lon:      u.evt.Point.Lon,
		Hae:      u.evt.Point.Hae,
		Icon:     GetIcon(u.Type),
	}

	if u.evt.Detail.Track != nil {
		w.Speed = u.evt.Detail.Track.Speed
	}

	if u.evt.Detail.Remarks != nil {
		w.Text = u.evt.Detail.Remarks.Text
	}

	if u.evt.Detail.Group != nil {
		w.Team = u.evt.Detail.Group.Name
		w.Role = u.evt.Detail.Group.Role
	}

	if v := u.evt.Detail.TakVersion; v != nil {
		w.TakVersion = strings.Trim(fmt.Sprintf("%s %s on %s", v.Platform, v.Version, v.Device), " ")
	}
	return w
}

func GetIcon(fn string) string {
	if !strings.HasPrefix(fn, "a-") {
		return ""
	}

	var prefix string

	switch fn[2] {
	case 'u':
		prefix = "0.su"
	case 'f':
		prefix = "1.sf"
	case 'n':
		prefix = "2.sn"
	case 'h':
		prefix = "3.sh"
	}

	return prefix + getCode(fn) + ".png"
}

func getCode(fn string) string {
	switch {
	case strings.HasPrefix(fn[4:], "G-E-V-A"):
		return "gpeva"
	case strings.HasPrefix(fn[4:], "G-E-V-C"):
		return "gpevc"

	case strings.HasPrefix(fn[4:], "A-C-F"), strings.HasPrefix(fn[4:], "G-C-F"):
		return "apcf"
	case strings.HasPrefix(fn[4:], "A-C-R"), strings.HasPrefix(fn[4:], "G-C-R"):
		return "apch"
	case strings.HasPrefix(fn[4:], "A-C"), strings.HasPrefix(fn[4:], "G-C"):
		return "apc"
	case strings.HasPrefix(fn[4:], "A-"):
		return "ap"

	default:
		return "gp"
	}
}
