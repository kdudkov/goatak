package model

import "testing"

func TestGetIcon(t *testing.T) {
	check(t, "a-u-G", "0.sugp.png")
	check(t, "a-f-G", "1.sfgp.png")
	check(t, "a-n-G", "2.sngp.png")
	check(t, "a-h-G", "3.shgp.png")

}

func check(t *testing.T, fn, icon string) {
	if GetIcon(fn) != icon {
		t.Errorf("got %s, must be %s", GetIcon(fn), icon)
	}
}
