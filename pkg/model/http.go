package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

type WebUnit struct {
	UID            string    `json:"uid"`
	Callsign       string    `json:"callsign"`
	Category       string    `json:"category"`
	Scope          string    `json:"scope"`
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
	Device         string    `json:"device"`
	Status         string    `json:"status"`
	Battery        uint32    `json:"battery"`
	Text           string    `json:"text"`
	Color          string    `json:"color"`
	Icon           string    `json:"icon"`
	ParentCallsign string    `json:"parent_callsign"`
	ParentUID      string    `json:"parent_uid"`
	Local          bool      `json:"local"`
	Send           bool      `json:"send"`
	Missions       []string  `json:"missions"`
}

type Contact struct {
	UID          string `json:"uid"`
	Callsign     string `json:"callsign"`
	Team         string `json:"team"`
	Role         string `json:"role"`
	Takv         string `json:"takv"`
	Notes        string `json:"notes"`
	FilterGroups string `json:"filterGroups"`
}

type DigitalPointer struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Name string  `json:"name"`
}

func (i *Item) ToWeb() *WebUnit {
	i.mx.RLock()
	defer i.mx.RUnlock()

	msg := i.msg

	if msg == nil {
		return nil
	}

	evt := msg.GetTakMessage().GetCotEvent()

	parentUID, parentCallsign := msg.GetParent()

	w := &WebUnit{
		UID:            i.uid,
		Category:       i.class,
		Scope:          msg.Scope,
		Callsign:       msg.GetCallsign(),
		Time:           cot.TimeFromMillis(evt.GetSendTime()),
		LastSeen:       i.lastSeen,
		StaleTime:      msg.GetStaleTime(),
		StartTime:      msg.GetStartTime(),
		SendTime:       msg.GetSendTime(),
		Type:           msg.GetType(),
		Lat:            evt.GetLat(),
		Lon:            evt.GetLon(),
		Hae:            evt.GetHae(),
		Speed:          evt.GetDetail().GetTrack().GetSpeed(),
		Course:         evt.GetDetail().GetTrack().GetCourse(),
		Team:           evt.GetDetail().GetGroup().GetName(),
		Role:           evt.GetDetail().GetGroup().GetRole(),
		Sidc:           getSIDC(msg.GetType()),
		ParentUID:      parentUID,
		ParentCallsign: parentCallsign,
		Color:          msg.GetColor(),
		Icon:           msg.GetIconsetPath(),
		Missions:       msg.GetDetail().GetDestMission(),
		Local:          i.local,
		Send:           i.send,
		Text:           msg.GetDetail().GetFirst("remarks").GetText(),
		TakVersion:     "",
		Battery:        msg.GetTakMessage().GetCotEvent().GetDetail().GetStatus().GetBattery(),
		Status:         "",
	}

	if i.class == CONTACT {
		if i.online {
			w.Status = "Online"
		} else {
			w.Status = "Offline"
		}

		if v := msg.GetTakv(); v != nil {
			w.TakVersion = strings.Trim(fmt.Sprintf("%s %s", v.GetPlatform(), v.GetVersion()), " ")
			w.Device = strings.Trim(fmt.Sprintf("%s, %s", v.GetDevice(), v.GetOs()), " ")
		}
	}

	return w
}

//nolint:exhaustruct
func (w *WebUnit) ToMsg() *cot.CotMessage {
	msg := &cotproto.TakMessage{
		CotEvent: &cotproto.CotEvent{
			Type:      w.Type,
			Uid:       w.UID,
			SendTime:  cot.TimeToMillis(w.SendTime),
			StartTime: cot.TimeToMillis(w.StartTime),
			StaleTime: cot.TimeToMillis(w.StaleTime),
			How:       "h-g-i-g-o",
			Lat:       w.Lat,
			Lon:       w.Lon,
			Hae:       w.Hae,
			Ce:        cot.NotNum,
			Le:        cot.NotNum,
			Detail: &cotproto.Detail{
				Contact: &cotproto.Contact{Callsign: w.Callsign},
				PrecisionLocation: &cotproto.PrecisionLocation{
					Geopointsrc: "USER",
					Altsrc:      "USER",
				},
			},
		},
	}

	xd := cot.NewXMLDetails()
	if w.ParentUID != "" {
		xd.AddPpLink(w.ParentUID, "", w.ParentCallsign)
	}

	xd.AddOrChangeChild("status", map[string]string{"readiness": "true"})

	if w.Text != "" {
		xd.AddChild("remarks", nil, w.Text)
	}

	msg.GetCotEvent().Detail.XmlDetail = xd.AsXMLString()

	zero := time.Unix(0, 0)

	if msg.GetCotEvent().GetUid() == "" {
		msg.CotEvent.Uid = uuid.New().String()
	}

	if w.StartTime.Before(zero) {
		msg.CotEvent.StartTime = cot.TimeToMillis(time.Now())
	}

	if w.SendTime.Before(zero) {
		msg.CotEvent.SendTime = cot.TimeToMillis(time.Now())
	}

	if w.StaleTime.Before(zero) {
		msg.CotEvent.StaleTime = cot.TimeToMillis(time.Now().Add(time.Hour * 24))
	}

	return &cot.CotMessage{
		From:       "",
		Scope:      w.Scope,
		TakMessage: msg,
		Detail:     xd,
	}
}

//nolint:gomnd
func getSIDC(fn string) string {
	if !strings.HasPrefix(fn, "a-") {
		return ""
	}

	tokens := strings.Split(fn, "-")

	sidc := "S" + tokens[1]

	if len(tokens) > 2 {
		sidc += tokens[2] + "P"
	} else {
		sidc += "-P"
	}

	if len(tokens) > 3 {
		for _, c := range tokens[3:] {
			if len(c) > 1 {
				break
			}

			sidc += c
		}
	}

	if len(sidc) < 12 {
		sidc += strings.Repeat("-", 10-len(sidc))
	}

	return strings.ToUpper(sidc)
}
