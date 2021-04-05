package cot

import (
	"reflect"
	"testing"

	v1 "github.com/kdudkov/goatak/cot/v1"
)

func TestConvert(t *testing.T) {
	msg := &v1.TakMessage{CotEvent: &v1.CotEvent{
		Type:      "a-f-G-U-C",
		Uid:       "ANDROID-55511111111555",
		SendTime:  1616961632610,
		StartTime: 1616961632610,
		StaleTime: 1616962007610,
		How:       "h-g-i-g-o",
		Hae:       9999999,
		Ce:        9999999,
		Le:        9999999,
		Detail: &v1.Detail{
			XmlDetail: "<uid Droid=\"callsign\"></uid>",
			Contact: &v1.Contact{
				Endpoint: "*:-1:stcp",
				Callsign: "callsign",
			},
			Group: &v1.Group{
				Name: "Red",
				Role: "Forward Observer",
			},
			Status: &v1.Status{Battery: 65},
			Takv: &v1.Takv{
				Device:   "Samsung",
				Platform: "ATAK-CIV",
				Os:       "29",
				Version:  "4.2.0.0 (0d581081).1608139612-CIV",
			},
			Track: &v1.Track{
				Speed:  55,
				Course: 11.1,
			},
		},
	}}

	evt := ProtoToEvent(msg)
	msg1 := EventToProto(evt)

	if !reflect.DeepEqual(msg.CotEvent, msg1.CotEvent) {
		t.Fail()
	}

}
