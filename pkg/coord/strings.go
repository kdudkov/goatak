package coord

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	r1 = regexp.MustCompile(`[xX]=?(?P<x>\d{5,})[;,\s]*[yY]=?(?P<y>\d{5,})`)
	r2 = regexp.MustCompile(`(?P<x>-?\d+\.\d+)[;,\s]*(?P<y>-?\d+\.\d+)`)
	r3 = regexp.MustCompile(`(?P<x>\d+\.\d+)([nNsS])[;,\s]*(?P<y>\d+\.\d+)([eEwW])`)
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

	if r2.MatchString(s) {
		res := r2.FindStringSubmatch(s)

		lat, err := strconv.ParseFloat(res[1], 64)

		if err != nil {
			return 0, 0, err
		}

		lon, err := strconv.ParseFloat(res[2], 64)

		if err != nil {
			return 0, 0, err
		}

		return lat, lon, nil
	}

	if r3.MatchString(s) {
		res := r3.FindStringSubmatch(s)

		lat, err := strconv.ParseFloat(res[1], 64)

		if err != nil {
			return 0, 0, err
		}

		if res[2] == "S" || res[2] == "s" {
			lat = -lat
		}

		lon, err := strconv.ParseFloat(res[3], 64)

		if err != nil {
			return 0, 0, err
		}

		if res[4] == "W" || res[4] == "w" {
			lon = -lon
		}

		return lat, lon, nil
	}

	return 0, 0, nil
}
