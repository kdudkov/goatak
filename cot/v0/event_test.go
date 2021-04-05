package v0

import (
	"encoding/xml"
	"testing"
)

func TestEventMarshal(t *testing.T) {

	ev := MakePing("123")
	dat, _ := xml.Marshal(ev)

	println(string(dat))
}
