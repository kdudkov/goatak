package coord

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testData struct {
	s    string
	x, y float64
}

func TestStringConvert(t *testing.T) {
	data := []testData{
		{"x5709130 y6648746", 51.49220977324127, 35.14007432073522},
		{"x5709130y6648746", 51.49220977324127, 35.14007432073522},
		{"X=5709130, y6648746", 51.49220977324127, 35.14007432073522},
		{"X=5709130,y6648746", 51.49220977324127, 35.14007432073522},
		{"51.49 35.14", 51.49, 35.14},
		{"51.49,  -35.14", 51.49, -35.14},
		{"51.49N  35.14E", 51.49, 35.14},
		{"51.49N,  35.14w", 51.49, -35.14},
	}

	for _, d := range data {
		lat, lon, err := StringToLatLon(d.s)
		assert.NoError(t, err)
		assert.Equal(t, d.x, lat)
		assert.Equal(t, d.y, lon)
	}
}
