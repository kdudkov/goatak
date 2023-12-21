package cot

import (
	_ "embed"
	"sort"
	"strings"
)

//go:embed types
var strTypes string

var (
	types = make(map[string]*CotType)
	Root  = new(CotType)
)

type CotType struct {
	Code string     `json:"code"`
	Name string     `json:"name"`
	Next []*CotType `json:"next"`
}

func init() {
	for _, s := range strings.Split(strTypes, "\n") {
		ss := strings.Trim(s, " \n\r\t")
		if ss == "" {
			continue
		}

		n := strings.Split(ss, ";")
		types[n[0]] = &CotType{
			Code: n[0],
			Name: n[1],
			Next: nil,
		}
	}

	for _, ct := range types {
		n := strings.Split(ct.Code, "-")
		if len(n) == 1 {
			Root.Next = append(Root.Next, ct)

			continue
		}

		found := false

		for i := len(n) - 1; i > 0; i-- {
			t1 := strings.Join(n[:i], "-")
			if ct1, ok := types[t1]; ok {
				found = true

				ct1.Next = append(ct1.Next, ct)

				break
			}
		}

		if !found {
			Root.Next = append(Root.Next, ct)
		}
	}

	for _, ct := range types {
		if ct.Next != nil {
			sort.SliceStable(ct.Next, func(i, j int) bool {
				return strings.Compare(ct.Next[i].Code, ct.Next[j].Code) < 0
			})
		}
	}

	sort.SliceStable(Root.Next, func(i, j int) bool {
		return strings.Compare(Root.Next[i].Code, Root.Next[j].Code) < 0
	})
}

func (t *CotType) Level() int {
	if t == nil {
		return 0
	}

	return strings.Count(t.Code, "-") + 1
}

func GetNext(s string) []*CotType {
	return types[s].Next
}
