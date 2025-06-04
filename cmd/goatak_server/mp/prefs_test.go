package mp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPref(t *testing.T) {
	for _,s := range []string{"locationTeam", "hidePreferenceItem_routePreference"} {
		e := GetEntry(s, "aaa")
		assert.NotEmpty(t, e)
	}
}
