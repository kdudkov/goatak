package cotxml

import (
	"encoding/xml"
)

type XMLDetail struct {
	XMLName  xml.Name  `xml:"detail"`
	Uid      *Uid      `xml:"uid,omitempty" `
	Contact  *Contact2 `xml:"contact,omitempty"`
	Status   *Status2  `xml:"status,omitempty"`
	Usericon *Usericon `xml:"usericon,omitempty"`
	Track    *Track    `xml:"track,omitempty"`
	Chat     *Chat     `xml:"__chat,omitempty"`
	Link     []*Link   `xml:"link,omitempty"`
	Remarks  *Remarks  `xml:"remarks,omitempty"`
	Marti    *Marti    `xml:"marti,omitempty"`

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

func XMLDetailFromString(s string) (*XMLDetail, error) {
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

func (d *XMLDetail) GetText() string {
	if d.Remarks != nil {
		return d.Remarks.Text
	}
	return ""
}
