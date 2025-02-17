package util

import "strings"

const sep = ';'

type StringSet map[string]any

func NewStringSet() StringSet {
	return make(StringSet)
}

func StringToSet(s string) StringSet {
	ss := make(StringSet)

	for _, s1 := range strings.Split(s, string(sep)) {
		ss.Add(s1)
	}

	return ss
}

func (s StringSet) Add(key string) {
	s[key] = struct{}{}
}

func (s StringSet) Remove(key string) {
	delete(s, key)
}

func (s StringSet) Has(key string) bool {
	_, ok := s[key]

	return ok
}

func (s StringSet) List() []string {
	res := make([]string, 0, len(s))

	for k := range s {
		res = append(res, k)
	}

	return res
}

func (s StringSet) String() string {
	sb := strings.Builder{}

	var notFirst bool

	for k := range s {
		if notFirst {
			sb.WriteRune(sep)
		} else {
			notFirst = true
		}

		sb.WriteString(k)
	}

	return sb.String()
}
