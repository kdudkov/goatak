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
}

func (u *Unit) ToWeb() *WebUnit {

	w := &WebUnit{
		Uid:      u.Uid,
		Callsign: u.Callsign,
		Time:     u.evt.Time,
		LastSeen: u.LastSeen,
		Stale:    u.Stale,
		Type:     u.Type,
		Lat:      u.evt.Point.Lat,
		Lon:      u.evt.Point.Lon,
		Hae:      u.evt.Point.Hae,
		Icon:     GetIcon(u),
		Sidc:     getSIDC(u.Type),
	}

	if u.evt.Detail.Track != nil {
		w.Speed = u.evt.Detail.Track.Speed
		w.Course = u.evt.Detail.Track.Course
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

func GetIcon(u *Unit) string {
	if !strings.HasPrefix(u.Type, "a-") {
		return ""
	}

	if g := u.evt.Detail.Group; g != nil {
		s := "roles/"

		switch g.Name {
		case "White":
			s += "white"
		case "Yellow":
			s += "grey"
		case "Orange":
			s += "grey"
		case "Magenta":
			s += "grey"
		case "Red":
			s += "red"
		case "Maroon":
			s += "grey"
		case "Purple":
			s += "grey"
		case "Dark Blue":
			s += "darkblue"
		case "Blue":
			s += "blue"
		case "Cyan":
			s += "cyan"
		case "Teal":
			s += "grey"
		case "Green":
			s += "green"
		case "Dark Green":
			s += "grey"
		case "Brown":
			s += "grey"
		default:
			s += "grey"
		}

		switch g.Role {
		case "Team Member":
		case "HQ":
			s += "-hq"
		case "Team Lead":
			s += "-tl"
		case "K9":
			s += "-k9"
		default:
		}
		return s + ".png"
	}

	return ""
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
