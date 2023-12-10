package coord

import (
	"fmt"
	"math"
	"testing"
)

func TestConvert(t *testing.T) {
	t.SkipNow()

	latt, lont := 60.000031779, 30.002262925
	xt, yt := 6657982.780, 6332761.808
	lat, lon := Wgs84_sk42(60, 30, 0)

	if math.Abs(lat-latt) > 0.000001 || math.Abs(lon-lont) > 0.000001 {
		t.Errorf("result: %f, %f", lat, lon)
	}

	x, y, z := Sk42ll2Meters(latt, lont)

	if math.Abs(x-xt) > 0.01 || math.Abs(y-yt) > 0.01 {
		t.Errorf("result: %f, %f %d", x, y, z)
	}

	fmt.Println(lat, lon)
	fmt.Println(x, y, z)
}

func TestConvertBoth(t *testing.T) {
	lat, lon := Wgs84_sk42(50, 50, 0)

	lat2, lon2 := Sk42_wgs84(lat, lon, 0)
	if math.Abs(lat2-50) > 0.0000001 {
		t.Errorf("result: %f, %f", lat2, lon2)
	}

	if math.Abs(lon2-50) > 0.0000001 {
		t.Errorf("result: %f, %f", lat2, lon2)
	}
}
