package cot

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/kdudkov/goatak/cotproto"
)

func ProtoToEvent(msg *cotproto.TakMessage) *Event {
	if msg == nil || msg.CotEvent == nil {
		return nil
	}

	ev := &Event{
		XMLName: xml.Name{Local: "event"},
		Version: "2.0",
		Type:    msg.CotEvent.Type,
		Uid:     msg.CotEvent.Uid,
		Time:    TimeFromMillis(msg.CotEvent.SendTime).UTC(),
		Start:   TimeFromMillis(msg.CotEvent.StartTime).UTC(),
		Stale:   TimeFromMillis(msg.CotEvent.StaleTime).UTC(),
		How:     msg.CotEvent.How,
		Point: Point{
			Lat: msg.CotEvent.Lat,
			Lon: msg.CotEvent.Lon,
			Hae: msg.CotEvent.Hae,
			Ce:  msg.CotEvent.Ce,
			Le:  msg.CotEvent.Le,
		},
		Detail: NewXmlDetails(),
	}

	if d := msg.CotEvent.Detail; d != nil {
		if d.XmlDetail != "" {
			b := bytes.Buffer{}
			b.WriteString("<detail>" + d.XmlDetail + "</detail>")
			xml.NewDecoder(&b).Decode(&ev.Detail)
			ev.Detail.Content = ""
		}

		if d.Contact != nil {
			attrs := map[string]string{
				"endpoint": d.Contact.Endpoint,
				"callsign": d.Contact.Callsign,
			}
			ev.Detail.AddChild("contact", attrs, "")
		}

		if d.Status != nil {
			ev.Detail.AddChild("status", map[string]string{"battery": strconv.Itoa(int(d.Status.Battery))}, "")
		}

		if d.Track != nil {
			attrs := map[string]string{
				"course": fmt.Sprintf("%f", d.Track.Course),
				"speed":  fmt.Sprintf("%f", d.Track.Speed),
			}
			ev.Detail.AddChild("track", attrs, "")
		}

		if d.Takv != nil {
			attrs := map[string]string{
				"os":       d.Takv.Os,
				"version":  d.Takv.Version,
				"device":   d.Takv.Device,
				"platform": d.Takv.Platform,
			}
			ev.Detail.AddChild("takv", attrs, "")
		}

		if d.Group != nil {
			attrs := map[string]string{
				"name": d.Group.Name,
				"role": d.Group.Role,
			}
			ev.Detail.AddChild("__group", attrs, "")
		}

		if d.PrecisionLocation != nil {
			attrs := map[string]string{
				"altsrc":      d.PrecisionLocation.Altsrc,
				"geopointsrc": d.PrecisionLocation.Geopointsrc,
			}
			ev.Detail.AddChild("precisionlocation", attrs, "")
		}
	}

	return ev
}

func EventToProto(ev *Event) (*cotproto.TakMessage, *Node) {
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

	if c := ev.Detail.GetFirst("contact"); c != nil {
		msg.CotEvent.Detail.Contact = &cotproto.Contact{
			Endpoint: c.GetAttr("endpoint"),
			Callsign: c.GetAttr("callsign"),
		}
	}

	if c := ev.Detail.GetFirst("__group"); c != nil {
		msg.CotEvent.Detail.Group = &cotproto.Group{
			Name: c.GetAttr("name"),
			Role: c.GetAttr("role"),
		}
	}

	if c := ev.Detail.GetFirst("precisionlocation"); c != nil {
		msg.CotEvent.Detail.PrecisionLocation = &cotproto.PrecisionLocation{
			Altsrc:      c.GetAttr("altsrc"),
			Geopointsrc: c.GetAttr("geopointsrc"),
		}
	}

	if c := ev.Detail.GetFirst("status"); c != nil {
		if n, err := strconv.Atoi(c.GetAttr("battery")); err == nil {
			msg.CotEvent.Detail.Status = &cotproto.Status{Battery: uint32(n)}
		}
	}

	if c := ev.Detail.GetFirst("takv"); c != nil {
		msg.CotEvent.Detail.Takv = &cotproto.Takv{
			Device:   c.GetAttr("device"),
			Platform: c.GetAttr("platform"),
			Os:       c.GetAttr("os"),
			Version:  c.GetAttr("version"),
		}
	}

	if c := ev.Detail.GetFirst("track"); c != nil {
		msg.CotEvent.Detail.Track = &cotproto.Track{
			Speed:  getFloat(c.GetAttr("speed")),
			Course: getFloat(c.GetAttr("course")),
		}
	}

	xd, _ := GetXmlDetails(ev.Detail)
	msg.CotEvent.Detail.XmlDetail = xd.AsXMLString()
	return msg, xd
}

func GetXmlDetails(d *Node) (*Node, error) {
	if d == nil {
		return nil, nil
	}

	b := bytes.Buffer{}
	if err := xml.NewEncoder(&b).Encode(d); err != nil {
		return nil, err
	}

	details, err := DetailsFromString(b.String())
	if err != nil {
		return nil, err
	}
	details.RemoveTags("contact", "__group", "precisionlocation", "status", "takv", "track")
	return details, nil
}

func getFloat(s string) float64 {
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err == nil {
		return f
	}
	return 0
}
