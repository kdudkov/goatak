package coord

import (
	"fmt"
	"math"
	"testing"
)

func TestConvert(t *testing.T) {
	lat, lon := WGS84_SK42(50, 50, 0)
	if lat != 49.999804136422554 || lon != 50.00152211915823 {
		t.Errorf("result: %f, %f", lat, lon)
	}
	fmt.Println(lat, lon)
}

func TestConvertBoth(t *testing.T) {
	lat, lon := WGS84_SK42(50, 50, 0)
	lat2, lon2 := SK42_WGS84(lat, lon, 0)
	if math.Abs(lat2-50) > 0.0000001 {
		t.Errorf("result: %f, %f", lat2, lon2)
	}

	if math.Abs(lon2-50) > 0.0000001 {
		t.Errorf("result: %f, %f", lat2, lon2)
	}
}
