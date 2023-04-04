package cot

import (
	"bufio"
	"bytes"
	"testing"
)

func TestProtoRW(t *testing.T) {
	msg := MakeDpMsg("testuid", "a-f-G", "test", 10, 20)

	b, err := MakeProtoPacket(msg)

	if err != nil {
		t.Fatal(err)
	}

	msg1, err := ReadProto(bufio.NewReader(bytes.NewReader(b)))

	if err != nil {
		t.Fatal(err)
	}

	if msg1.GetCotEvent().Lat != 10 {
		t.Fail()
	}
}
