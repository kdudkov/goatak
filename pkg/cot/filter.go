package cot

import "strings"

func MatchPattern(a, pattern string) bool {
	if strings.HasPrefix(a, pattern) && strings.HasSuffix(pattern, "-") {
		return true
	}

	at := strings.Split(a, "-")
	pt := strings.Split(strings.TrimRight(pattern, "-"), "-")

	if len(at) < len(pt) {
		return false
	}

	for i, s := range pt {
		if s != at[i] && s != "." {
			return false
		}
	}

	if strings.HasSuffix(pattern, "-") {
		return len(at) > len(pt)
	} else {
		return len(at) == len(pt)
	}
}
