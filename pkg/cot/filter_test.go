package cot

import "testing"

func TestMatch(t *testing.T) {
	data := []struct {
		a, b string
		res  bool
	}{
		{"a-b-c-d", ".-", true},
		{"a-b-c-d", "a-b-c", false},
		{"a-b-c-d", "a-b-c-", true},
		{"a-b-c-d", "a-.-c", false},
		{"a-b-c-d", "a-.-c-", true},
		{"a-b-c-d", ".-.-c-", true},
		{"a-b-c-d", "a-.-c-d", true},
		{"a-b-c-d", "a-b-c-d-", false},
		{"a-b-c-d", ".-b-c-d-", false},
	}

	for _, d := range data {
		if MatchPattern(d.a, d.b) != d.res {
			t.Errorf("%s %s", d.a, d.b)
		}
	}
}
