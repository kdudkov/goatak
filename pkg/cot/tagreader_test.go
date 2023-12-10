package cot

import (
	"fmt"
	"strings"
	"testing"
)

var data = []struct {
	name string
	data string
	tags []string
}{
	{
		name: "test1",
		data: "<?xml ?>sdfsdfsdf sdfsdf<aa>ccc</aa> <bbb parm1=2/><event> aaa <bb>asdasd</bb>asasd </event> ffff",
		tags: []string{"?xml", "aa", "bbb", "event"},
	},
	{
		name: "test2",
		data: "<event parm1=\"21\"> aaa <bb>asdasd</bb>asasd </event>",
		tags: []string{"event"},
	},
	{
		name: "test3",
		data: "sadfasdfasdfas</aaa><event parm1=\"21\"> aaa <bb>asdasd</bb>asasd </event>",
		tags: []string{"", "event"},
	},
}

func TestTagReader(t *testing.T) {
	for _, dat := range data {
		t.Run(dat.name, func(t *testing.T) {
			b := strings.NewReader(dat.data)

			reader := NewTagReader(b)
			for _, rightTag := range dat.tags {
				tag, dat, err := reader.ReadTag()
				fmt.Printf("%s %s\n", tag, dat)

				if rightTag != "" && err != nil {
					t.Error(err)
				}

				if tag != rightTag {
					t.Errorf("bad tag %s, muste be %s", tag, rightTag)
				}
			}
		})
	}
}
