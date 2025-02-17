package util

func FirstString(s ...string) string {
	for _, ss := range s {
		if ss != "" {
			return ss
		}
	}

	return ""
}
