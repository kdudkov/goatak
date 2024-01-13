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
	if msg == nil {
		return nil
	}

	return CotToEvent(msg.GetCotEvent())
}

func CotToEvent(c *cotproto.CotEvent) *Event {
	if c == nil {
		return nil
	}

	ev := &Event{
		XMLName: xml.Name{Local: "event"},
		Version: "2.0",
		Type:    c.GetType(),
		Access:  c.GetAccess(),
		Qos:     c.GetQos(),
		Opex:    c.GetOpex(),
		UID:     c.GetUid(),
		Time:    TimeFromMillis(c.GetSendTime()).UTC(),
		Start:   TimeFromMillis(c.GetStartTime()).UTC(),
		Stale:   TimeFromMillis(c.GetStaleTime()).UTC(),
		How:     c.GetHow(),
		Point: Point{
			Lat: c.GetLat(),
			Lon: c.GetLon(),
			Hae: c.GetHae(),
			Ce:  c.GetCe(),
			Le:  c.GetLe(),
		},
		Detail: NewXMLDetails(),
	}

	if d := c.GetDetail(); d != nil {
		if d.GetXmlDetail() != "" {
			b := bytes.Buffer{}
			b.WriteString("<detail>" + d.GetXmlDetail() + "</detail>")
			_ = xml.NewDecoder(&b).Decode(&ev.Detail)
		}

		if d.GetContact().GetCallsign() != "" {
			attrs := map[string]string{
				"callsign": d.GetContact().GetCallsign(),
				"endpoint": d.GetContact().GetEndpoint(),
			}
			ev.Detail.AddOrChangeChild("contact", attrs)
		}

		if d.GetStatus() != nil {
			ev.Detail.AddOrChangeChild("status", map[string]string{"battery": strconv.Itoa(int(d.GetStatus().GetBattery()))})
		}

		if d.GetTrack() != nil {
			attrs := map[string]string{
				"course": fmt.Sprintf("%f", d.GetTrack().GetCourse()),
				"speed":  fmt.Sprintf("%f", d.GetTrack().GetSpeed()),
			}
			ev.Detail.AddOrChangeChild("track", attrs)
		}

		if tv := d.GetTakv(); tv != nil {
			attrs := map[string]string{
				"os":       tv.GetOs(),
				"version":  tv.GetVersion(),
				"device":   tv.GetDevice(),
				"platform": tv.GetPlatform(),
			}
			ev.Detail.AddOrChangeChild("takv", attrs)
		}

		if d.GetGroup() != nil {
			attrs := map[string]string{
				"name": d.GetGroup().GetName(),
				"role": d.GetGroup().GetRole(),
			}
			ev.Detail.AddOrChangeChild("__group", attrs)
		}

		if d.GetPrecisionLocation() != nil {
			attrs := map[string]string{
				"altsrc":      d.GetPrecisionLocation().GetAltsrc(),
				"geopointsrc": d.GetPrecisionLocation().GetGeopointsrc(),
			}
			ev.Detail.AddOrChangeChild("precisionlocation", attrs)
		}
	}

	return ev
}

//nolint:nilnil
func EventToProto(ev *Event) (*CotMessage, error) {
	if ev == nil {
		return nil, nil
	}

	var extContact bool

	msg := &cotproto.TakMessage{CotEvent: &cotproto.CotEvent{
		Type:      ev.Type,
		Access:    ev.Access,
		Qos:       ev.Qos,
		Opex:      ev.Opex,
		Uid:       ev.UID,
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
		if len(c.GetAttrs()) < 3 {
			msg.CotEvent.Detail.Contact = &cotproto.Contact{
				Endpoint: c.GetAttr("endpoint"),
				Callsign: c.GetAttr("callsign"),
			}
		} else {
			extContact = true
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

	xd, err := GetXMLDetails(ev.Detail, extContact)
	msg.CotEvent.Detail.XmlDetail = xd.AsXMLString()

	return &CotMessage{TakMessage: msg, Detail: xd}, err
}

func EventToProtoExt(ev *Event, from, scope string) (*CotMessage, error) {
	c, err := EventToProto(ev)
	if c != nil {
		c.Scope = scope
		c.From = from
	}

	return c, err
}

//nolint:nilnil
func GetXMLDetails(d *Node, withContacts bool) (*Node, error) {
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

	removeFields := []string{"contact", "__group", "precisionlocation", "status", "takv", "track"}

	if withContacts {
		details.RemoveTags(removeFields[1:]...)
	} else {
		details.RemoveTags(removeFields...)
	}

	return details, nil
}

func getFloat(s string) float64 {
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err == nil {
		return f
	}

	return 0
}
