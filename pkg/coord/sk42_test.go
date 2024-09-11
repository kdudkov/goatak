package coord

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestData struct {
	lat, lon float64
	x, y     int
}

func TestConvert(t *testing.T) {
	data := []TestData{
		{57.712277, 33.643766, 6399533, 6538495},
		{50.0, 36.200553, 5544706, 7299419},
	}

	for _, d := range data {
		x, y, _ := Wgs84_sk42(d.lat, d.lon, 0)

		assert.InDelta(t, d.x, x, 1)
		assert.InDelta(t, d.y, y, 1)

		lat1, lon1 := Sk42_wgs(d.x, d.y)

		assert.InDelta(t, d.lat, lat1, 0.000005)
		assert.InDelta(t, d.lon, lon1, 0.000005)
	}

}
