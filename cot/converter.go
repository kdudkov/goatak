package cot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kdudkov/goatak/cotproto"
	"github.com/kdudkov/goatak/cotxml"
)

func ProtoToEvent(msg *cotproto.TakMessage) *cotxml.Event {
	if msg == nil || msg.CotEvent == nil {
		return nil
	}

	ev := &cotxml.Event{
		Version: "2.0",
		Type:    msg.CotEvent.Type,
		Uid:     msg.CotEvent.Uid,
		Time:    TimeFromMillis(msg.CotEvent.SendTime),
		Start:   TimeFromMillis(msg.CotEvent.StartTime),
		Stale:   TimeFromMillis(msg.CotEvent.StaleTime),
		How:     msg.CotEvent.How,
		Point: cotxml.Point{
			Lat: msg.CotEvent.Lat,
			Lon: msg.CotEvent.Lon,
			Hae: msg.CotEvent.Hae,
			Ce:  msg.CotEvent.Ce,
			Le:  msg.CotEvent.Le,
		},
	}

	if d := msg.CotEvent.Detail; d != nil {
		if d.Contact != nil {
			ev.Detail.Contact = &cotxml.Contact{
				Endpoint: d.Contact.Endpoint,
				Callsign: d.Contact.Callsign,
			}
		}

		if d.Status != nil {
			ev.Detail.Status = &cotxml.Status{
				Battery: strconv.Itoa(int(d.Status.Battery)),
			}
		}

		if d.Track != nil {
			ev.Detail.Track = &cotxml.Track{
				Course: fmt.Sprintf("%f", d.Track.Course),
				Speed:  fmt.Sprintf("%f", d.Track.Speed),
			}
		}

		if d.Takv != nil {
			ev.Detail.TakVersion = &cotxml.TakVersion{
				Os:       d.Takv.Os,
				Version:  d.Takv.Version,
				Device:   d.Takv.Device,
				Platform: d.Takv.Platform,
			}
		}

		if d.Group != nil {
			ev.Detail.Group = &cotxml.Group{
				Name: d.Group.Name,
				Role: d.Group.Role,
			}
		}

		if d.PrecisionLocation != nil {
			ev.Detail.PrecisionLocation = &cotxml.Precisionlocation{
				Altsrc:      d.PrecisionLocation.Altsrc,
				Geopointsrc: d.PrecisionLocation.Geopointsrc,
			}
		}

		if d.XmlDetail != "" {
			xd, err := cotxml.XMLDetailFromString(d.XmlDetail)
			if err == nil {
				applyDetails(&ev.Detail, xd)
			}
		}
	}

	return ev
}

func EventToProto(ev *cotxml.Event) (*cotproto.TakMessage, *cotxml.XMLDetail) {
	if ev == nil {
		return nil, nil
	}

	msg := &cotproto.TakMessage{CotEvent: &cotproto.CotEvent{
		Type:      ev.Type,
		Uid:       ev.Uid,
		SendTime:  TimeToMillis(ev.Time),
		StartTime: TimeToMillis(ev.Start),
		StaleTime: TimeToMillis(ev.Stale),
		How:       ev.How,
		Lat:       ev.Point.Lat,
		Lon:       ev.Point.Lon,
		Hae:       ev.Point.Hae,
		Ce:        ev.Point.Ce,
		Le:        ev.Point.Le,
		Detail:    &cotproto.Detail{},
	}}

	if c := ev.Detail.TakVersion; c != nil {
		msg.CotEvent.Detail.Takv = &cotproto.Takv{
			Device:   c.Device,
			Platform: c.Platform,
			Os:       c.Os,
			Version:  c.Version,
		}
	}

	if c := ev.Detail.Contact; c != nil {
		msg.CotEvent.Detail.Contact = &cotproto.Contact{
			Endpoint: c.Endpoint,
			Callsign: c.Callsign,
		}
	}

	if c := ev.Detail.PrecisionLocation; c != nil {
		msg.CotEvent.Detail.PrecisionLocation = &cotproto.PrecisionLocation{
			Geopointsrc: c.Geopointsrc,
			Altsrc:      c.Altsrc,
		}
	}

	if c := ev.Detail.Group; c != nil {
		msg.CotEvent.Detail.Group = &cotproto.Group{
			Name: c.Name,
			Role: c.Role,
		}
	}

	if c := ev.Detail.Status; c != nil {
		if n, err := strconv.Atoi(c.Battery); err == nil {
			msg.CotEvent.Detail.Status = &cotproto.Status{Battery: uint32(n)}
		}
	}

	if c := ev.Detail.Track; c != nil {
		msg.CotEvent.Detail.Track = &cotproto.Track{
			Speed:  getFloat(c.Speed),
			Course: getFloat(c.Course),
		}
	}

	xd := GetXmlDetails(&ev.Detail)
	msg.CotEvent.Detail.XmlDetail = xd.String()

	return msg, xd
}

func GetXmlDetails(d *cotxml.Detail) *cotxml.XMLDetail {
	if d == nil {
		return nil
	}

	d1 := &cotxml.XMLDetail{
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
		d1.Contact = &cotxml.Contact2{Phone: d.Contact.Phone}
	}

	if d.Status != nil && d.Status.Readiness != "" {
		d1.Status = &cotxml.Status2{Readiness: d.Status.Readiness}
	}

	return d1
}

func applyDetails(d1 *cotxml.Detail, xd *cotxml.XMLDetail) {
	if xd == nil {
		return
	}

	if xd.Contact != nil {
		if d1.Contact == nil {
			d1.Contact = &cotxml.Contact{}
		}
		d1.Contact.Phone = xd.Contact.Phone
	}

	if xd.Status != nil {
		if d1.Status == nil {
			d1.Status = &cotxml.Status{}
		}
		d1.Status.Readiness = xd.Status.Readiness
	}

	d1.Uid = xd.Uid
	d1.StrokeWeight = xd.StrokeWeight
	d1.Color = xd.Color
	d1.FillColor = xd.FillColor
	d1.StrokeColor = xd.StrokeColor
	d1.Marti = xd.Marti
	d1.Chat = xd.Chat
}

func getFloat(s string) float64 {
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err == nil {
		return f
	}
	return 0
}
