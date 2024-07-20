package main

import (
	"fmt"
	"strings"

	"github.com/kdudkov/goatak/pkg/cot"
)

type ContactsDumper struct {
	res map[string][]string
}

func (d *ContactsDumper) Start() {
	d.res = make(map[string][]string)
}

func (d *ContactsDumper) Stop() {
	for _, uid := range sortedKeys(d.res) {
		fmt.Printf("%s: %s\n", uid, strings.Join(d.res[uid], ", "))
	}
}

func (d *ContactsDumper) Process(msg *cot.CotMessage) error {
	if !msg.IsContact() {
		return nil
	}

	uid := msg.GetUID()
	cs := msg.GetCallsign()

	if l, ok := d.res[uid]; ok {
		for _, s := range l {
			if s == cs {
				return nil
			}
		}
		d.res[uid] = append(l, cs)
	} else {
		d.res[uid] = []string{cs}
	}

	return nil
}
