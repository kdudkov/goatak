package cot

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/kdudkov/goatak/pkg/cotproto"
)

func ProtoToEvent(msg *cotproto.TakMessage) *Event {
	if msg == nil || msg.GetCotEvent() == nil {
		return nil
	}

	ev := &Event{
		XMLName: xml.Name{Local: "event"},
		Version: "2.0",
		Type:    msg.GetCotEvent().GetType(),
		Access:  msg.GetCotEvent().GetAccess(),
		Qos:     msg.GetCotEvent().GetQos(),
		Opex:    msg.GetCotEvent().GetOpex(),
		Uid:     msg.GetCotEvent().GetUid(),
		Time:    TimeFromMillis(msg.GetCotEvent().GetSendTime()).UTC(),
		Start:   TimeFromMillis(msg.GetCotEvent().GetStartTime()).UTC(),
		Stale:   TimeFromMillis(msg.GetCotEvent().GetStaleTime()).UTC(),
		How:     msg.GetCotEvent().GetHow(),
		Point: Point{
			Lat: msg.GetCotEvent().GetLat(),
			Lon: msg.GetCotEvent().GetLon(),
			Hae: msg.GetCotEvent().GetHae(),
			Ce:  msg.GetCotEvent().GetCe(),
			Le:  msg.GetCotEvent().GetLe(),
		},
		Detail: NewXmlDetails(),
	}

	if d := msg.GetCotEvent().GetDetail(); d != nil {
		if d.GetXmlDetail() != "" {
			b := bytes.Buffer{}
			b.WriteString("<detail>" + d.GetXmlDetail() + "</detail>")
			xml.NewDecoder(&b).Decode(&ev.Detail)
		}

		if d.GetContact() != nil {
			attrs := map[string]string{
				"endpoint": d.GetContact().GetEndpoint(),
				"callsign": d.GetContact().GetCallsign(),
			}
			ev.Detail.AddChild("contact", attrs, "")
		}

		if d.GetStatus() != nil {
			ev.Detail.AddChild("status", map[string]string{"battery": strconv.Itoa(int(d.GetStatus().GetBattery()))}, "")
		}

		if d.GetTrack() != nil {
			attrs := map[string]string{
				"course": fmt.Sprintf("%f", d.GetTrack().GetCourse()),
				"speed":  fmt.Sprintf("%f", d.GetTrack().GetSpeed()),
			}
			ev.Detail.AddChild("track", attrs, "")
		}

		if tv := d.GetTakv(); tv != nil {
			attrs := map[string]string{
				"os":       tv.GetOs(),
				"version":  tv.GetVersion(),
				"device":   tv.GetDevice(),
				"platform": tv.GetPlatform(),
			}
			ev.Detail.AddChild("takv", attrs, "")
		}

		if d.GetGroup() != nil {
			attrs := map[string]string{
				"name": d.GetGroup().GetName(),
				"role": d.GetGroup().GetRole(),
			}
			ev.Detail.AddChild("__group", attrs, "")
		}

		if d.GetPrecisionLocation() != nil {
			attrs := map[string]string{
				"altsrc":      d.GetPrecisionLocation().GetAltsrc(),
				"geopointsrc": d.GetPrecisionLocation().GetGeopointsrc(),
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
		Access:    ev.Access,
		Qos:       ev.Qos,
		Opex:      ev.Opex,
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
