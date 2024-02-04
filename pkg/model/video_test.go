package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestData struct {
	url   string
	proto string
	addr  string
	port  int
	path  string
}

func TestParse(t *testing.T) {
	data := []TestData{
		{"https://exapmle.com", "https", "exapmle.com", 443, ""},
		{"https://exapmle.com/aa/bb", "https", "exapmle.com", 443, "/aa/bb"},
		{"https://exapmle.com:8080/asdasdasd/dddd/sss?aa", "https", "exapmle.com", 8080, "/asdasdasd/dddd/sss?aa"},
		{"rtsp://exapmle.com/aa/bb", "rtsp", "exapmle.com", 554, "/aa/bb"},
		{"srt://exapmle.com/aa/bb", "srt", "exapmle.com", 9710, "/aa/bb"},
	}

	for _, td := range data {
		t.Run("parse_"+td.url, func(t *testing.T) {
			proto, addr, port, path := parseURL(td.url)

			if proto != td.proto || addr != td.addr || port != td.port || path != td.path {
				t.Errorf("%s -> %s %s %d %s", td.url, proto, addr, port, path)
			}

			newURL := toURL(proto, addr, port, path)

			assert.Equal(t, td.url, newURL)
		})
	}
}
