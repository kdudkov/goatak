package v1

import (
	"encoding/xml"

	v0 "github.com/kdudkov/goatak/cot/v0"
)

type XMLDetail struct {
	XMLName  xml.Name     `xml:"detail"`
	Uid      *v0.Uid      `xml:"uid,omitempty" `
	Contact  *Contact2    `xml:"contact,omitempty"`
	Status   *Status2     `xml:"status,omitempty"`
	Usericon *v0.Usericon `xml:"usericon,omitempty"`
	Track    *Track       `xml:"track,omitempty"`
	Chat     *v0.Chat     `xml:"__chat,omitempty"`
	Link     []*v0.Link   `xml:"link,omitempty"`
	Remarks  *v0.Remarks  `xml:"remarks,omitempty"`
	Marti    *v0.Marti    `xml:"marti,omitempty"`

	Color *struct {
		Value string `xml:"argb,attr,omitempty"`
	} `xml:"color,omitempty" v1:"full"`
	StrokeColor *struct {
		Value string `xml:"value,attr,omitempty"`
	} `xml:"strokeColor,omitempty" v1:"full"`
	FillColor *struct {
		Value string `xml:"value,attr,omitempty"`
	} `xml:"fillColor,omitempty" v1:"full"`
	StrokeWeight *struct {
		Value string `xml:"value,attr,omitempty"`
	} `xml:"strokeWeight,omitempty" v1:"full"`
}
type Contact2 struct {
	Phone string `xml:"phone,attr,omitempty"`
}

type Status2 struct {
	Text      string `xml:",chardata"`
	Readiness string `xml:"readiness,attr,omitempty"`
}

func (d *XMLDetail) String() string {
	if b, err := xml.Marshal(d); err == nil {
		s := string(b)
		if len(s) > 17 {
			return s[8 : len(s)-9]
		}
	}

	return ""
}

func FromString(s string) (*XMLDetail, error) {
	d := &XMLDetail{}

	if s == "" {
		return d, nil
	}

	if err := xml.Unmarshal([]byte("<detail>"+s+"</detail>"), d); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *XMLDetail) GetCallsignTo() []string {
	if d.Marti != nil {
		res := make([]string, len(d.Marti.Dest))
		for i, d := range d.Marti.Dest {
			res[i] = d.Callsign
		}
		return res
	}
	return nil
}
