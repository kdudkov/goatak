package main

import (
	"archive/zip"
	"bytes"
	"fmt"
)

type ZipFile interface {
	SetName(name string)
	Name() string
	Content() []byte
}

type PrefFile struct {
	name string
	data map[string]any
}

func NewUserProfilePrefFile() *PrefFile {
	return NewPrefFile("user-profile.pref")
}

func NewDefaultPrefFile() *PrefFile {
	f := NewPrefFile("defaults.pref")
	f.AddParam("speed_unit_pref", "1")
	f.AddParam("alt_unit_pref", "1")
	f.AddBoolParam("saHasPhoneNumber", false)
	f.AddParam("locationUnitType", "a-f-G-U-C")
	f.AddParam("alt_display_pref", "MSL")
	f.AddParam("coord_display_pref", "DM")
	f.AddParam("rab_north_ref_pref", "1")
	f.AddParam("rab_brg_units", "0")
	f.AddParam("rab_nrg_units", "1")
	return f
}

func NewPrefFile(name string) *PrefFile {
	return &PrefFile{name: name, data: make(map[string]any)}
}

func (p *PrefFile) AddParam(k, v string) {
	p.data[k] = v
}

func (p *PrefFile) AddBoolParam(k string, v bool) {
	p.data[k] = v
}

func (p *PrefFile) Name() string {
	return p.name
}

func (p *PrefFile) SetName(name string) {
	p.name = name
}

func (p *PrefFile) Content() []byte {
	var sb bytes.Buffer
	sb.WriteString("<?xml version='1.0' standalone='yes'?>\n")
	sb.WriteString("<preferences>\n<preference version=\"1\" name=\"com.atakmap.app.civ_preferences\">\n")
	for k, v := range p.data {
		var cl string
		switch v.(type) {
		case bool:
			cl = "class java.lang.Boolean"
		default:
			cl = "class java.lang.String"
		}
		sb.WriteString(fmt.Sprintf("<entry key=\"%s\" class=\"%s\">%v</entry>", k, cl, v))
	}
	sb.WriteString("</preference></preferences>")
	return sb.Bytes()
}

type MissionPackage struct {
	params map[string]string
	files  []ZipFile
}

func NewMissionPackage(uuid, name string) *MissionPackage {
	return &MissionPackage{params: map[string]string{"uid": uuid, "name": name}}
}

func (m *MissionPackage) Param(k, v string) {
	m.params[k] = v
}

func (m *MissionPackage) AddFile(f ZipFile) {
	m.files = append(m.files, f)
}

func (m *MissionPackage) Manifest() []byte {
	buf := bytes.Buffer{}
	buf.WriteString("<MissionPackageManifest version=\"2\">\n")
	buf.WriteString("<Configuration>")
	for k, v := range m.params {
		buf.WriteString(fmt.Sprintf("<Parameter name=\"%s\" value=\"%s\"/>", k, v))
	}
	buf.WriteString("</Configuration>")
	buf.WriteString("<Contents>")
	for _, v := range m.files {
		buf.WriteString(fmt.Sprintf("<Content ignore=\"false\" zipEntry=\"%s\"/>", v))
	}
	buf.WriteString("</Contents>")
	return buf.Bytes()
}

func (m *MissionPackage) Create() ([]byte, error) {
	var buff bytes.Buffer
	zipW := zip.NewWriter(&buff)

	f, err := zipW.Create("MANIFEST/manifest.xml")

	if err != nil {
		return nil, err
	}

	_, _ = f.Write(m.Manifest())

	for _, zf := range m.files {
		f1, err := zipW.Create(zf.Name())
		if err != nil {
			return nil, err
		}

		_, _ = f1.Write(zf.Content())
	}

	zipW.Close()
	return buff.Bytes(), nil
}
