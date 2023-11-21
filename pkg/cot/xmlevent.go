package cot

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"time"
)

type Event struct {
	XMLName xml.Name  `xml:"event"`
	Version string    `xml:"version,attr"`
	Type    string    `xml:"type,attr"`
	Access  string    `xml:"access,attr,omitempty"`
	Qos     string    `xml:"qos,attr,omitempty"`
	Opex    string    `xml:"opex,attr,omitempty"`
	Uid     string    `xml:"uid,attr"`
	Time    time.Time `xml:"time,attr"`
	Start   time.Time `xml:"start,attr"`
	Stale   time.Time `xml:"stale,attr"`
	How     string    `xml:"how,attr"`
	Detail  *Node     `xml:"detail"`
	Point   Point     `xml:"point"`
}

func (e *Event) String() string {
	if e == nil {
		return "nil"
	}
	return fmt.Sprintf("version=%s, type=%s, uid=%s, how=%s, stale=%s, detail={%s}", e.Version, e.Type, e.Uid, e.How, e.Stale.Sub(e.Start), e.Detail)
}

func (e *Event) AddDetail() *Node {
	if e == nil {
		return nil
	}

	if e.Detail == nil {
		e.Detail = &Node{XMLName: xml.Name{Local: "detail"}}
	}

	return e.Detail
}

func (e *Event) AddGroup(group, role string) {
	if e == nil {
		return
	}

	e.AddDetail().AddOrChangeChild("__group", map[string]string{"name": group, "role": role})
}

func (e *Event) AddCallsign(callsign, endpoint string, addDroid bool) {
	if e == nil {
		return
	}

	e.AddDetail().AddOrChangeChild("contact", map[string]string{"callsign": callsign, "endpoint": endpoint})

	if addDroid {
		e.AddDetail().AddOrChangeChild("uid", map[string]string{"Droid": callsign})
	}
}

func (e *Event) AddTrack(speed, course string) {
	if e == nil {
		return
	}

	e.AddDetail().AddOrChangeChild("track", map[string]string{"speed": speed, "course": course})
}

func (e *Event) AddVersion(device, platform, os, version string) {
	if e == nil {
		return
	}

	e.AddDetail().AddOrChangeChild("takv", map[string]string{"device": device, "platform": platform, "os": os, "version": version})
}

type Point struct {
	XMLName xml.Name `xml:"point"`
	Lat     float64  `xml:"lat,attr"`
	Lon     float64  `xml:"lon,attr"`
	Hae     float64  `xml:"hae,attr"`
	Ce      float64  `xml:"ce,attr"`
	Le      float64  `xml:"le,attr"`
}

func XmlBasicMsg(typ string, uid string, stale time.Duration) *Event {
	return &Event{
		Version: "2.0",
		Uid:     uid,
		Type:    typ,
		Time:    time.Now().UTC(),
		Start:   time.Now().UTC(),
		Stale:   time.Now().Add(stale).UTC(),
		Point: Point{
			Lat: 0,
			Lon: 0,
			Hae: 0,
			Ce:  9999999,
			Le:  9999999,
		},
	}
}

func VersionSupportMsg(ver int8) *Event {
	v := strconv.Itoa(int(ver))
	ev := XmlBasicMsg("t-x-takp-v", "protouid", time.Minute)
	ev.How = "m-g"
	ev.AddDetail().AddOrChangeChild("TakControl", nil).AddOrChangeChild("TakProtocolSupport", map[string]string{"version": v})
	return ev
}

func VersionReqMsg(ver int8) *Event {
	v := strconv.Itoa(int(ver))
	ev := XmlBasicMsg("t-x-takp-q", "protouid", time.Minute)
	ev.How = "m-g"
	ev.AddDetail().AddOrChangeChild("TakControl", nil).AddOrChangeChild("TakRequest", map[string]string{"version": v})
	return ev
}

func ProtoChangeOkMsg() *Event {
	ev := XmlBasicMsg("t-x-takp-r", "protouid", time.Minute)
	ev.How = "m-g"
	ev.AddDetail().AddOrChangeChild("TakControl", nil).AddOrChangeChild("TakResponse", map[string]string{"status": "true"})
	return ev
}

// Geopointsrc = "USER" Altsrc - "DTED0"
// ce
// high   0 - cat1,  7 - CAT2 16 - CAT3 31 - CAT4 92 - CAT5
// medium 3 - cat1, 11 - CAT2 23 - CAT3 61 - CAT4 198.5 - CAT5
// low    6 - cat1, 15 - CAT2 30 - CAT3 91 - CAT4 305 - CAT5
// UNKNOWN 9999999
