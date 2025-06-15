package mp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdudkov/goatak/pkg/util"
)

func TestGetPref(t *testing.T) {
	for _, s := range []string{"locationTeam", "hidePreferenceItem_routePreference"} {
		e := GetEntry(s, "aaa")
		assert.NotEmpty(t, e)
	}
}

func TestDouble(t *testing.T) {
	s := util.NewStringSet()

	for _, p := range prefKeys {
		if s.Has(p.Key) {
			t.Errorf(" dublicate key %s", p.Key)
		}

		s.Add(p.Key)
	}
}
