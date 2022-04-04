package model

import (
	"testing"
)

func TestSiDC(t *testing.T) {
	checkSIDC(t, "a-u-G", "SUGP------")
	checkSIDC(t, "a-f-G-U-C", "SFGPUC----")
	checkSIDC(t, "a-n-A-C-F", "SNAPCF----")
	checkSIDC(t, "a-f-G-wasp-struct", "SFGP------")
}

func checkSIDC(t *testing.T, fn, sidc string) {
	if getSIDC(fn) != sidc {
		t.Errorf("got %s, must be %s", getSIDC(fn), sidc)
	}
}
