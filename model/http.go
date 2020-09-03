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
		Time:     u.Evt.Time,
		LastSeen: u.LastSeen,
		Stale:    u.Stale,
		Type:     u.Type,
		Lat:      u.Evt.Point.Lat,
		Lon:      u.Evt.Point.Lon,
		Hae:      u.Evt.Point.Hae,
		Icon:     GetIcon(u),
		Sidc:     getSIDC(u.Type),
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

	if v := u.Evt.Detail.TakVersion; v != nil {
		w.TakVersion = strings.Trim(fmt.Sprintf("%s %s on %s", v.Platform, v.Version, v.Device), " ")
	}
	return w
}

func GetIcon(u *Unit) string {
	if !strings.HasPrefix(u.Type, "a-") {
		return ""
	}

	if g := u.Evt.Detail.Group; g != nil {
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
