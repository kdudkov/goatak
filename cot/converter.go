package cot

import (
	"encoding/xml"
	"strconv"

	v0 "github.com/kdudkov/goatak/cot/v0"
	v1 "github.com/kdudkov/goatak/cot/v1"
)

func ProtoToEvent(msg *v1.TakMessage) *v0.Event {
	if msg == nil || msg.CotEvent == nil {
		return nil
	}

	ev := &v0.Event{
		Version: "2.0",
		Type:    msg.CotEvent.Type,
		Uid:     msg.CotEvent.Uid,
		Time:    v1.TimeFromMillis(msg.CotEvent.SendTime),
		Start:   v1.TimeFromMillis(msg.CotEvent.StartTime),
		Stale:   v1.TimeFromMillis(msg.CotEvent.StaleTime),
		How:     msg.CotEvent.How,
		Point: v0.Point{
			Lat: msg.CotEvent.Lat,
			Lon: msg.CotEvent.Lon,
			Hae: msg.CotEvent.Hae,
			Ce:  msg.CotEvent.Ce,
			Le:  msg.CotEvent.Le,
		},
	}

	if d := msg.CotEvent.Detail; d != nil {
		if d.Contact != nil {
			ev.Detail.Contact = &v0.Contact{
				Endpoint: d.Contact.Endpoint,
				Callsign: d.Contact.Callsign,
			}
		}

		if d.Status != nil {
			ev.Detail.Status = &v0.Status{
				Battery: strconv.Itoa(int(d.Status.Battery)),
			}
		}

		if d.Track != nil {
			ev.Detail.Track = &v0.Track{
				Course: d.Track.Course,
				Speed:  d.Track.Speed,
			}
		}

		if d.Takv != nil {
			ev.Detail.TakVersion = &v0.TakVersion{
				Os:       d.Takv.Os,
				Version:  d.Takv.Version,
				Device:   d.Takv.Device,
				Platform: d.Takv.Platform,
			}
		}

		if d.Group != nil {
			ev.Detail.Group = &v0.Group{
				Name: d.Group.Name,
				Role: d.Group.Role,
			}
		}

		if d.PrecisionLocation != nil {
			ev.Detail.PrecisionLocation = &v0.Precisionlocation{
				Altsrc:      d.PrecisionLocation.Altsrc,
				Geopointsrc: d.PrecisionLocation.Geopointsrc,
			}
		}

		if d.XmlDetail != "" {
			d2 := &v0.Detail{}
			if err := xml.Unmarshal([]byte("<detail>"+d.XmlDetail+"</detail>"), d2); err == nil {
				applyDetails(&ev.Detail, d2)
			}
		}
	}

	return ev
}

func EventToProto(ev *v0.Event) (*v1.TakMessage, *v1.XMLDetail) {
	if ev == nil {
		return nil, nil
	}

	msg := &v1.TakMessage{CotEvent: &v1.CotEvent{
		Type:      ev.Type,
		Uid:       ev.Uid,
		SendTime:  v1.TimeToMillis(ev.Time),
		StartTime: v1.TimeToMillis(ev.Start),
		StaleTime: v1.TimeToMillis(ev.Stale),
		How:       ev.How,
		Lat:       ev.Point.Lat,
		Lon:       ev.Point.Lon,
		Hae:       ev.Point.Hae,
		Ce:        ev.Point.Ce,
		Le:        ev.Point.Le,
		Detail:    &v1.Detail{},
	}}

	if c := ev.Detail.TakVersion; c != nil {
		msg.CotEvent.Detail.Takv = &v1.Takv{
			Device:   c.Device,
			Platform: c.Platform,
			Os:       c.Os,
			Version:  c.Version,
		}
	}

	if c := ev.Detail.Contact; c != nil {
		msg.CotEvent.Detail.Contact = &v1.Contact{
			Endpoint: c.Endpoint,
			Callsign: c.Callsign,
		}
	}

	if c := ev.Detail.PrecisionLocation; c != nil {
		msg.CotEvent.Detail.PrecisionLocation = &v1.PrecisionLocation{
			Geopointsrc: c.Geopointsrc,
			Altsrc:      c.Altsrc,
		}
	}

	if c := ev.Detail.Group; c != nil {
		msg.CotEvent.Detail.Group = &v1.Group{
			Name: c.Name,
			Role: c.Role,
		}
	}

	if c := ev.Detail.Status; c != nil {
		if n, err := strconv.Atoi(c.Battery); err == nil {
			msg.CotEvent.Detail.Status = &v1.Status{Battery: uint32(n)}
		}
	}

	if c := ev.Detail.Track; c != nil {
		msg.CotEvent.Detail.Track = &v1.Track{
			Speed:  c.Speed,
			Course: c.Course,
		}
	}

	xd := GetXmlDetails(&ev.Detail)

	if xd != nil {
		if b, err := xml.Marshal(xd); err == nil {
			s := string(b)
			if len(s) > 17 {
				msg.CotEvent.Detail.XmlDetail = s[8 : len(s)-9]
			}
		}
	}
	return msg, xd
}

func GetXmlDetails(d *v0.Detail) *v1.XMLDetail {
	if d == nil {
		return nil
	}

	d1 := &v1.XMLDetail{
		Uid:          d.Uid,
		Usericon:     d.Usericon,
		Chat:         d.Chat,
		Link:         d.Link,
		Remarks:      d.Remarks,
		Marti:        d.Marti,
		Color:        d.Color,
		StrokeColor:  d.StrokeColor,
		FillColor:    d.FillColor,
		StrokeWeight: d.StrokeWeight,
	}

	if d.Contact != nil && d.Contact.Phone != "" {
		d1.Contact = &v1.Contact2{Phone: d.Contact.Phone}
	}

	if d.Status != nil && d.Status.Readiness != "" {
		d1.Status = &v1.Status2{Readiness: d.Status.Readiness}
	}

	return d1
}

func applyDetails(d1, d2 *v0.Detail) {
	if d2 == nil {
		return
	}

	if d2.Contact != nil {
		if d1.Contact == nil {
			d1.Contact = &v0.Contact{}
		}
		d1.Contact.Phone = d2.Contact.Phone
	}

	if d2.Status != nil {
		if d1.Status == nil {
			d1.Status = &v0.Status{}
		}
		d1.Status.Readiness = d2.Status.Readiness
	}

	d1.Uid = d2.Uid
	d1.StrokeWeight = d2.StrokeWeight
	d1.Color = d2.Color
	d1.FillColor = d2.FillColor
	d1.StrokeColor = d2.StrokeColor
	d1.Marti = d2.Marti
	d1.Chat = d2.Chat
}
