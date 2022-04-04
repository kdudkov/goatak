package cot

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/kdudkov/goatak/cotproto"
)

func TestUnmarshal(t *testing.T) {
	t.Skip()
	b, err := ioutil.ReadFile("./msg")
	if err != nil {
		t.Errorf("error %v", err)
	}

	evt := new(cotproto.TakMessage)
	if b[0] == 0xbf {
		l, n1 := binary.Uvarint(b[1:])
		fmt.Println(l, n1)
		err = proto.Unmarshal(b[n1+1:], evt)

		if err != nil {
			t.Errorf("error %v", err)
		}
	}
}

func TestMarshal(t *testing.T) {
	evt := &cotproto.TakMessage{
		CotEvent: &cotproto.CotEvent{
			Type:      "aaaaa",
			Access:    "",
			Qos:       "",
			Opex:      "",
			Uid:       "",
			SendTime:  0,
			StartTime: 0,
			StaleTime: 0,
			How:       "",
			Lat:       0,
			Lon:       0,
			Hae:       0,
			Ce:        0,
			Le:        0,
			Detail:    nil,
		},
	}

	bytes, err := proto.Marshal(evt)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	fmt.Println(len(bytes))
}
