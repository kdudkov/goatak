package cot

import (
	"testing"

	"github.com/kdudkov/goatak/xml"
)

func TestEventMarshal(t *testing.T) {

	ev := MakePing("123")
	dat, _ := xml.Marshal(ev)

	println(string(dat))
}
