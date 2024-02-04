package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSiDC(t *testing.T) {
	checkSIDC(t, "a-u-G", "SUGP------")
	checkSIDC(t, "a-f-G-U-C", "SFGPUC----")
	checkSIDC(t, "a-n-A-C-F", "SNAPCF----")
	checkSIDC(t, "a-f-G-wasp-struct", "SFGP------")
}

func checkSIDC(t *testing.T, fn, sidc string) {
	t.Helper()
	assert.Equal(t, sidc, getSIDC(fn))
}
