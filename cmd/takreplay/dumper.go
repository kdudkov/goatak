package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kdudkov/goatak/pkg/cot"
)

type Dumper interface {
	Start()
	Stop()
	Process(msg *cot.CotMessage) error
}

type TextDumper struct{}

func (g *TextDumper) Start() {
}

func (g *TextDumper) Stop() {
}

func (g *TextDumper) Process(msg *cot.CotMessage) error {
	fmt.Println(msg.GetSendTime().Format(time.DateTime), msg.GetUID(), msg.GetType(), msg.GetCallsign(), cot.GetMsgType(msg.GetType()))

	return nil
}

type JsonDumper struct{}

func (g *JsonDumper) Start() {
}

func (g *JsonDumper) Stop() {
}

func (g *JsonDumper) Process(msg *cot.CotMessage) error {
	b, err := json.Marshal(msg.GetTakMessage())
	if err != nil {
		return err
	}

	fmt.Println(string(b))

	return nil
}

type Json2Dumper struct{}

func (g *Json2Dumper) Start() {
}

func (g *Json2Dumper) Stop() {
}

func (g *Json2Dumper) Process(msg *cot.CotMessage) error {
	b, err := json.Marshal(msg.GetTakMessage())
	if err != nil {
		return err
	}

	fmt.Println(string(b))
	fmt.Println(msg.GetTakMessage().GetCotEvent().GetDetail().GetXmlDetail())

	return nil
}

type GpxDumper struct {
	name       string
	prevStale  time.Time
	hasHistory bool
}

func (g *GpxDumper) Start() {
	fmt.Println("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	fmt.Println("<gpx\nxmlns=\"http://www.topografix.com/GPX/1/1\"\nversion=\"1.1\"\ncreator=\"takreplay\"\nxmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\"\nxsi:schemaLocation=\"http://www.topografix.com/GPX/1/1 http://www.topografix.com/GPX/1/1/gpx.xsd\">")
	fmt.Printf("<metadata><name>Track 1</name></metadata>\n")

	fmt.Printf("<trk><name>%s</name>\n<trkseg>\n", g.name)
}

func (g *GpxDumper) Stop() {
	fmt.Println("</trkseg></trk></gpx>")
}

func (g *GpxDumper) Process(msg *cot.CotMessage) error {
	if msg == nil || msg.GetTakMessage().GetCotEvent() == nil || (msg.GetTakMessage().GetCotEvent().GetLat() == 0 && msg.GetTakMessage().GetCotEvent().GetLon() == 0) {
		return nil
	}

	ev := msg.GetTakMessage().GetCotEvent()

	if g.hasHistory && msg.GetStartTime().After(g.prevStale) {
		fmt.Println("</trkseg>\n<trkseg>")
	}

	fmt.Printf("<trkpt lat=\"%f\" lon=\"%f\">", ev.GetLat(), ev.GetLon())
	fmt.Printf("<time>%s</time>", msg.GetStartTime().Format(time.RFC3339))
	fmt.Printf("<ele>%.0f</ele>", ev.GetHae())
	fmt.Printf("<fix>%.0f</fix>", ev.GetLe())
	fmt.Printf("</trkpt>\n")

	g.prevStale = msg.GetStaleTime()
	g.hasHistory = true

	return nil
}
