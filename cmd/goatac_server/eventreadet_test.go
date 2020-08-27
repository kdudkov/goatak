package main

import (
	"strings"
	"testing"
)

func TestEventreader_ReadEvent1(t *testing.T) {
	b := strings.NewReader("<?xml ?><aa>ccc</aa> <event> aaa <bb>asdasd</bb>asasd </event> ffff")

	er := NewEventnReader(b)
	dat, err := er.ReadEvent()
	if err != nil {
		t.Error(err)
	}

	if string(dat) != "<event> aaa <bb>asdasd</bb>asasd </event>" {
		t.Error(string(dat))
	}
}

func TestEventreader_ReadEvent2(t *testing.T) {
	b := strings.NewReader("<aa>ccc</aa> <event parm1=\"21\"> aaa <bb>asdasd</bb>asasd </event> ffff")

	er := NewEventnReader(b)
	dat, err := er.ReadEvent()
	if err != nil {
		t.Error(err)
	}

	if string(dat) != "<event parm1=\"21\"> aaa <bb>asdasd</bb>asasd </event>" {
		t.Error(string(dat))
	}
}

func TestEventreader_ReadEvent3(t *testing.T) {
	b := strings.NewReader("<adasdasd<event parm1=\"21\"> aaa <bb>asdasd</bb>asasd </event> ffff")

	er := NewEventnReader(b)
	dat, err := er.ReadEvent()
	if err != nil {
		t.Error(err)
	}

	if string(dat) != "<event parm1=\"21\"> aaa <bb>asdasd</bb>asasd </event>" {
		t.Error(string(dat))
	}
}
