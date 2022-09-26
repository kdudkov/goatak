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
	Uid     string    `xml:"uid,attr"`
	Time    time.Time `xml:"time,attr"`
	Start   time.Time `xml:"start,attr"`
	Stale   time.Time `xml:"stale,attr"`
	How     string    `xml:"how,attr"`

	Point  Point `xml:"point"`
	Detail *Node `xml:"detail"`
}

func (e *Event) String() string {
	if e == nil {
		return "nil"
	}
	return fmt.Sprintf("version=%s, type=%s, uid=%s, how=%s, stale=%s, detail={%s}", e.Version, e.Type, e.Uid, e.How, e.Stale.Sub(e.Start), e.Detail)
}

type Point struct {
	XMLName xml.Name `xml:"point"`
	Lat     float64  `xml:"lat,attr"`
	Lon     float64  `xml:"lon,attr"`
	Hae     float64  `xml:"hae,attr"`
	Ce      float64  `xml:"ce,attr"`
	Le      float64  `xml:"le,attr"`
}

func (e *Event) IsTakControlRequest() bool {
	return e.Detail.GetFirst("TakControl").GetFirst("TakRequest") != nil
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
	ev.Detail = NewXmlDetails()
	ev.Detail.AddChild("TakControl", nil, "").AddChild("TakProtocolSupport", map[string]string{"version": v}, "")
	return ev
}

func VersionReqMsg(ver int8) *Event {
	v := strconv.Itoa(int(ver))
	ev := XmlBasicMsg("t-x-takp-v", "protouid", time.Minute)
	ev.How = "m-g"
	ev.Detail = NewXmlDetails()
	ev.Detail.AddChild("TakControl", nil, "").AddChild("TakRequest", map[string]string{"version": v}, "")
	return ev
}

func ProtoChangeOkMsg() *Event {
	ev := XmlBasicMsg("t-x-takp-r", "protouid", time.Minute)
	ev.How = "m-g"
	ev.Detail = NewXmlDetails()
	ev.Detail.AddChild("TakControl", nil, "").AddChild("TakResponse", map[string]string{"status": "true"}, "")
	return ev
}

// Geopointsrc = "USER" ce Altsrc - "DTED0"
// high   0 - cat1,  7 - CAT2 16 - CAT3 31 - CAT4 92 - CAT5
// medium 3 - cat1, 11 - CAT2 23 - CAT3 61 - CAT4 198.5 - CAT5
// low    6 - cat1, 15 - CAT2 30 - CAT3 91 - CAT4 305 - CAT5
// UNKNOWN 9999999
