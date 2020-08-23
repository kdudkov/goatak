package cot

import (
	"testing"

	"gotac/xml"
)

func TestEventMarshal(t *testing.T) {

	ev := MakePing("123")
	dat, _ := xml.Marshal(ev)

	println(string(dat))
}
