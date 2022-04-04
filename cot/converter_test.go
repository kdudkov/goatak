package cot

import (
	"reflect"
	"testing"

	"github.com/kdudkov/goatak/cotproto"
)

func TestConvert(t *testing.T) {
	msg := &cotproto.TakMessage{CotEvent: &cotproto.CotEvent{
		Type:      "a-f-G-U-C",
		UID:       "ANDROID-55511111111555",
		SendTime:  1616961632610,
		StartTime: 1616961632610,
		StaleTime: 1616962007610,
		How:       "h-g-i-g-o",
		Hae:       9999999,
		Ce:        9999999,
		Le:        9999999,
		Detail: &cotproto.Detail{
			XmlDetail: "<uid Droid=\"callsign\"></uid>",
			Contact: &cotproto.Contact{
				Endpoint: "*:-1:stcp",
				Callsign: "callsign",
			},
			Group: &cotproto.Group{
				Name: "Red",
				Role: "Forward Observer",
			},
			Status: &cotproto.Status{Battery: 65},
			Takv: &cotproto.Takv{
				Device:   "Samsung",
				Platform: "ATAK-CIV",
				Os:       "29",
				Version:  "4.2.0.0 (0d581081).1608139612-CIV",
			},
			Track: &cotproto.Track{
				Speed:  55,
				Course: 11.1,
			},
		},
	}}

	evt := ProtoToEvent(msg)
	msg1, _ := EventToProto(evt)

	if !reflect.DeepEqual(msg.CotEvent, msg1.CotEvent) {
		t.Fail()
	}
}
