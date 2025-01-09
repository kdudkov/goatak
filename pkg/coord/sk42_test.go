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
		{51.214357, 35.257187, 5678463, 6657824},
	}

	for _, d := range data {
		x, y, _ := Wgs84_sk42(d.lat, d.lon, 0)

		assert.InDelta(t, d.x, x, 5, "bad x")
		assert.InDelta(t, d.y, y, 5, "bad y")

		lat1, lon1 := Sk42_wgs(d.x, d.y)

		assert.InDelta(t, d.lat, lat1, 0.00005, "bad lat")
		assert.InDelta(t, d.lon, lon1, 0.00009, "bad lon")
	}

}
