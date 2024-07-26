package main

import (
	"fmt"
	"strings"

	"github.com/kdudkov/goatak/pkg/cot"
)

type StatsDumper struct {
	data     map[string]int64
	devices  map[string]int64
	versions map[string]int64
}

func (g *StatsDumper) Start() {
	g.data = make(map[string]int64)
	g.devices = make(map[string]int64)
	g.versions = make(map[string]int64)
}

func (g *StatsDumper) Stop() {
	fmt.Println("\n== Messages:")

	for k, v := range g.data {
		fmt.Printf("%s %s %d\n", k, cot.GetMsgType(k), v)
	}

	fmt.Println("\n== Devices:")

	for _, k := range sortedKeys(g.devices) {
		fmt.Printf("%s\n", k)
	}

	fmt.Println("\n== Versions:")

	for _, k := range sortedKeys(g.versions) {
		fmt.Printf("%s\n", k)
	}
}

func (g *StatsDumper) Process(msg *cot.CotMessage) error {
	t := msg.GetType()

	if strings.HasPrefix(t, "a-") && len(t) > 5 {
		t = t[:5]
	}

	if n, ok := g.data[t]; ok {
		g.data[t] = n + 1
	} else {
		g.data[t] = 1
	}

	if v := msg.GetTakv(); v != nil {
		var vv string
		if strings.IndexByte(v.GetVersion(), '\n') >= 0 {
			vv, _, _ = strings.Cut(v.GetVersion(), "\n")
		} else {
			vv = v.GetVersion()
		}

		ver := strings.Trim(fmt.Sprintf("%s %s", v.GetPlatform(), vv), " ")
		dev := strings.Trim(fmt.Sprintf("%s (%s)", v.GetDevice(), v.GetOs()), " ")

		if !strings.Contains(ver, "\n") {
			if n, ok := g.versions[ver]; ok {
				g.versions[ver] = n + 1
			} else {
				g.versions[ver] = 1
			}
		}

		if n, ok := g.devices[dev]; ok {
			g.devices[dev] = n + 1
		} else {
			g.devices[dev] = 1
		}
	}

	return nil
}
