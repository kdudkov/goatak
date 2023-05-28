package main

import "encoding/xml"

type VideoConnections struct {
	XMLName xml.Name
	Feeds   []*Feed `xml:"feed"`
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
}
