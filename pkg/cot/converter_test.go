package cot

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"testing"

	"github.com/kdudkov/goatak/pkg/cotproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert(t *testing.T) {
	msg := &cotproto.TakMessage{CotEvent: &cotproto.CotEvent{
		Type:      "a-f-G-U-C",
		Uid:       "ANDROID-55511111111555",
		SendTime:  1616961632610,
		StartTime: 1616961632610,
		StaleTime: 1616962007610,
		How:       "h-g-i-g-o",
		Hae:       NotNum,
		Ce:        NotNum,
		Le:        NotNum,
		Detail: &cotproto.Detail{
			XmlDetail: "<uid Droid=\"callsign\"></uid><remarks>remark text</remarks>",
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

	b, _ := xml.Marshal(evt)
	fmt.Println(string(b))

	cot, _ := EventToProto(evt)

	assert.Equal(t, "<uid Droid=\"callsign\"></uid><remarks>remark text</remarks>", cot.TakMessage.GetCotEvent().GetDetail().GetXmlDetail())

	if !reflect.DeepEqual(msg.GetCotEvent(), cot.TakMessage.GetCotEvent()) {
		t.Fail()
	}
}

func TestConvert2(t *testing.T) {
	msg := &cotproto.TakMessage{CotEvent: &cotproto.CotEvent{
		Type:      "a-f-G-U-C",
		Uid:       "ANDROID-55511111111555",
		SendTime:  1616961632610,
		StartTime: 1616961632610,
		StaleTime: 1616962007610,
		How:       "h-g-i-g-o",
		Hae:       NotNum,
		Ce:        NotNum,
		Le:        NotNum,
		Detail: &cotproto.Detail{
			XmlDetail: "<contact callsign=\"callsign\" endpoint=\"*:-1:stcp\" phone=\"555\"></contact><uid Droid=\"callsign\"></uid><remarks>remark text</remarks>",
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

	cot1, err := CotFromProto(msg, "", "")
	require.NoError(t, err)

	assert.Equal(t, "callsign", cot1.GetCallsign())
	assert.Equal(t, "*:-1:stcp", cot1.GetEndpoint())

	evt := ProtoToEvent(msg)

	b, _ := xml.Marshal(evt)
	fmt.Println(string(b))

	cot, _ := EventToProto(evt)

	assert.Equal(t, "<contact callsign=\"callsign\" endpoint=\"*:-1:stcp\" phone=\"555\"></contact><uid Droid=\"callsign\"></uid><remarks>remark text</remarks>", cot.TakMessage.GetCotEvent().GetDetail().GetXmlDetail())

	if !reflect.DeepEqual(msg.GetCotEvent(), cot.TakMessage.GetCotEvent()) {
		t.Fail()
	}
}
