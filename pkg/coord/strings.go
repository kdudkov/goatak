package coord

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	r1 = regexp.MustCompile(`[xX](?P<x>\d{5,}),?\s+[yY](?P<y>\d{5,})`)
)

func StringToLatLon(s string) (float64, float64, error) {
	s = strings.Trim(s, " \t\n\r.,")

	if r1.MatchString(s) {
		res := r1.FindStringSubmatch(s)

		x, err := strconv.Atoi(res[1])

		if err != nil {
			return 0, 0, err
		}

		y, err := strconv.Atoi(res[2])

		if err != nil {
			return 0, 0, err
		}

		lat, lon := Sk42_wgs(x, y)

		return lat, lon, nil
	}

	return 0, 0, nil
}
