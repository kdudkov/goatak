package coord

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSK42(t *testing.T) {
	lat, lon, err := StringToLatLon("x5709130 y6648746")

	assert.NoError(t, err)

	fmt.Println(lat, lon)
}
