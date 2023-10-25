package model

import (
	"encoding/xml"
	"fmt"
	url2 "net/url"
	"strconv"
	"strings"
)

var defports = map[string]int{
	"http":  80,
	"https": 443,
	"rtsp":  554,
	"rtmp":  1935,
	"srt":   9710,
}

type VideoConnections struct {
	XMLName xml.Name
	Feeds   []*Feed `xml:"feed"`
}

type VideoConnections2 struct {
	XMLName xml.Name `json:"-"`
	Feeds   []*Feed2 `xml:"feed" json:"feeds"`
}

type Feed struct {
	Uid    string `xml:"uid" yaml:"uid"`
	Active bool   `xml:"active" yaml:"active,omitempty"`
	Alias  string `xml:"alias" yaml:"alias"`
	Typ    string `xml:"type" yaml:"type,omitempty"`

	Address             string  `xml:"address" yaml:"address,omitempty"`
	Path                string  `xml:"path" yaml:"path,omitempty"`
	PreferredMacAddress string  `xml:"preferredMacAddress" yaml:"preferredMacAddress,omitempty"`
	Port                int     `xml:"port" yaml:"port,omitempty"`
	RoverPort           int     `xml:"roverPort" yaml:"roverPort,omitempty"`
	IgnoreEmbeddedKLV   bool    `xml:"ignoreEmbeddedKLV" yaml:"ignoreEmbeddedKLV,omitempty"`
	Protocol            string  `xml:"protocol" yaml:"protocol,omitempty"`
	Source              string  `xml:"source" yaml:"source,omitempty"`
	Timeout             int     `xml:"timeout" yaml:"timeout,omitempty"`
	Buffer              int     `xml:"buffer" yaml:"buffer,omitempty"`
	RtspReliable        string  `xml:"rtspReliable" yaml:"rtspReliable,omitempty"`
	Thumbnail           string  `xml:"thumbnail" yaml:"thumbnail,omitempty"`
	Classification      string  `xml:"classification" yaml:"classification,omitempty"`
	Latitude            float64 `xml:"latitude" yaml:"latitude,omitempty"`
	Longitude           float64 `xml:"longitude" yaml:"longitude,omitempty"`
	Fov                 string  `xml:"fov" yaml:"fov,omitempty"`
	Heading             string  `xml:"heading" yaml:"heading,omitempty"`
	Range               string  `xml:"range" yaml:"range,omitempty"`
	User                string  `yaml:"user"`
}

type Feed2 struct {
	Uid       string  `yaml:"uid" json:"uid,omitempty"`
	Active    bool    `yaml:"active" json:"active"`
	Alias     string  `yaml:"alias" json:"alias,omitempty"`
	Url       string  `yaml:"url" json:"url,omitempty"`
	Latitude  float64 `yaml:"lat,omitempty" json:"lat,omitempty"`
	Longitude float64 `yaml:"lon,omitempty" json:"lon,omitempty"`
	Fov       string  `yaml:"fov,omitempty" json:"fov,omitempty"`
	Heading   string  `yaml:"heading,omitempty" json:"heading,omitempty"`
	Range     string  `yaml:"range,omitempty" json:"range,omitempty"`
	User      string  `yaml:"user" json:"-"`
}

func (f *Feed2) ToFeed() *Feed {
	if f == nil {
		return nil
	}

	proto, addr, port, path := parseUrl(f.Url)

	return &Feed{
		Uid:                 f.Uid,
		Active:              true,
		Alias:               f.Alias,
		Typ:                 "",
		Address:             addr,
		Path:                path,
		PreferredMacAddress: "",
		Port:                port,
		RoverPort:           -1,
		IgnoreEmbeddedKLV:   false,
		Protocol:            proto,
		Source:              "",
		Timeout:             12000,
		Buffer:              -1,
		RtspReliable:        "",
		Thumbnail:           "",
		Classification:      "",
		Latitude:            f.Latitude,
		Longitude:           f.Longitude,
		Fov:                 f.Fov,
		Heading:             f.Heading,
		Range:               f.Range,
		User:                f.User,
	}
}

func (f *Feed2) WithUser(user string) *Feed2 {
	f.User = user
	return f
}

func (f *Feed) ToFeed2() *Feed2 {
	if f == nil {
		return nil
	}

	return &Feed2{
		Active:    f.Active,
		Uid:       f.Uid,
		Alias:     f.Alias,
		Url:       toUrl(f.Protocol, f.Address, f.Port, f.Path),
		Latitude:  f.Latitude,
		Longitude: f.Longitude,
		Fov:       f.Fov,
		Heading:   f.Heading,
		Range:     f.Range,
		User:      f.User,
	}
}

func parseUrl(url string) (proto, addr string, port int, path string) {
	u, err := url2.Parse(url)
	if err != nil {
		return
	}

	proto = u.Scheme
	addr = u.Host
	path = u.Path

	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}

	if i := strings.Index(u.Host, ":"); i > 0 {
		addr = u.Host[:i]
		port, _ = strconv.Atoi(u.Host[i+1:])
	} else {
		port = defports[proto]
	}

	return
}

func toUrl(proto, addr string, port int, path string) string {
	if p, ok := defports[proto]; (ok && p == port) || port == 0 {
		return fmt.Sprintf("%s://%s%s", proto, addr, path)
	} else {
		return fmt.Sprintf("%s://%s:%d%s", proto, addr, port, path)
	}
}
