package model

import "testing"

func TestSiDC(t *testing.T) {
	checkSIDC(t, "a-u-G", "SUG-------")
	checkSIDC(t, "a-f-G-U-C", "SFG-UC----")
	checkSIDC(t, "a-n-A-C-F", "SNA-CF----")
}

func checkSIDC(t *testing.T, fn, sidc string) {
	if getSIDC(fn) != sidc {
		t.Errorf("got %s, must be %s", getSIDC(fn), sidc)
	}
}
