//nolint:gomnd
package model

import (
	"encoding/xml"
	"fmt"
	url2 "net/url"
	"strconv"
	"strings"
)

var defPorts = map[string]int{
	"http":  80,
	"https": 443,
	"rtsp":  554,
	"rtmp":  1935,
	"srt":   9710,
}

type VideoConnections struct {
	XMLName xml.Name   `xml:"videoConnections"`
	Feeds   []*FeedDTO `xml:"feed"`
}

type VideoConnections2 struct {
	XMLName xml.Name    `json:"-"`
	Feeds   []*Feed2DTO `json:"feeds" xml:"feed"`
}

type FeedDTO struct {
	UID                 string  `xml:"uid"`
	Active              bool    `xml:"active"`
	Alias               string  `xml:"alias"`
	Typ                 string  `xml:"type"`
	Address             string  `xml:"address"`
	Path                string  `xml:"path"`
	PreferredMacAddress string  `xml:"preferredMacAddress"`
	Port                int     `xml:"port"`
	RoverPort           int     `xml:"roverPort"`
	IgnoreEmbeddedKLV   bool    `xml:"ignoreEmbeddedKLV"`
	Protocol            string  `xml:"protocol"`
	Source              string  `xml:"source"`
	Timeout             int     `xml:"timeout"`
	Buffer              int     `xml:"buffer"`
	RtspReliable        string  `xml:"rtspReliable"`
	Thumbnail           string  `xml:"thumbnail"`
	Classification      string  `xml:"classification"`
	Latitude            float64 `xml:"latitude"`
	Longitude           float64 `xml:"longitude"`
	Fov                 string  `xml:"fov"`
	Heading             string  `xml:"heading"`
	Range               string  `xml:"range"`
}

type Feed2 struct {
	UID       string `gorm:"primaryKey;size:255"`
	Active    bool
	Alias     string `gorm:"size:255"`
	URL       string `gorm:"size:512"`
	Latitude  float64
	Longitude float64
	Fov       string `gorm:"size:100"`
	Heading   string `gorm:"size:100"`
	Range     string `gorm:"size:100"`
	User      string `gorm:"size:255;index"`
	Scope     string `gorm:"size:255"`
}

type Feed2DTO struct {
	UID       string  `json:"uid,omitempty"`
	Active    bool    `json:"active"`
	Alias     string  `json:"alias,omitempty"`
	URL       string  `json:"url,omitempty"`
	Latitude  float64 `json:"lat,omitempty"`
	Longitude float64 `json:"lon,omitempty"`
	Fov       string  `json:"fov,omitempty"`
	Heading   string  `json:"heading,omitempty"`
	Range     string  `json:"range,omitempty"`
	User      string  `json:"user,omitempty"`
	Scope     string  `json:"scope,omitempty"`
}

type FeedPutDTO struct {
	Active    bool    `json:"active"`
	Alias     string  `json:"alias,omitempty"`
	URL       string  `json:"url,omitempty"`
	Latitude  float64 `json:"lat,omitempty"`
	Longitude float64 `json:"lon,omitempty"`
	Fov       string  `json:"fov,omitempty"`
	Heading   string  `json:"heading,omitempty"`
	Range     string  `json:"range,omitempty"`
	Scope     string  `json:"scope,omitempty"`
}

type FeedPostDTO struct {
	UID string `json:"uid,omitempty"`
	FeedPutDTO
}

func (f *Feed2) TableName() string {
	return "feeds"
}

func (f *Feed2) DTOOld() *FeedDTO {
	if f == nil {
		return nil
	}

	proto, addr, port, path := parseURL(f.URL)

	return &FeedDTO{
		UID:                 f.UID,
		Active:              f.Active,
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
	}
}

func (f *Feed2) DTO(admin bool) *Feed2DTO {
	if f == nil {
		return nil
	}

	dto := &Feed2DTO{
		UID:       f.UID,
		Active:    f.Active,
		Alias:     f.Alias,
		URL:       f.URL,
		Latitude:  f.Latitude,
		Longitude: f.Longitude,
		Fov:       f.Fov,
		Heading:   f.Heading,
		Range:     f.Range,
	}

	if admin {
		dto.User = f.User
		dto.Scope = f.Scope
	}

	return dto
}

func (f *FeedDTO) ToFeed2() *Feed2 {
	if f == nil {
		return nil
	}

	return &Feed2{
		Active:    f.Active,
		UID:       f.UID,
		Alias:     f.Alias,
		URL:       toURL(f.Protocol, f.Address, f.Port, f.Path),
		Latitude:  f.Latitude,
		Longitude: f.Longitude,
		Fov:       f.Fov,
		Heading:   f.Heading,
		Range:     f.Range,
	}
}

//nolint:nonamedreturns
func parseURL(url string) (proto, addr string, port int, path string) {
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
		port = defPorts[proto]
	}

	return
}

func toURL(proto, addr string, port int, path string) string {
	if p, ok := defPorts[proto]; (ok && p == port) || port == 0 {
		return fmt.Sprintf("%s://%s%s", proto, addr, path)
	}

	return fmt.Sprintf("%s://%s:%d%s", proto, addr, port, path)
}
