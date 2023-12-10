package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"strings"
)

type MissionPackage struct {
	params map[string]string
	files  []FileContent
}

func NewMissionPackage(uuid, name string) *MissionPackage {
	return &MissionPackage{params: map[string]string{"uid": uuid, "name": name}}
}

func (m *MissionPackage) Param(k, v string) {
	m.params[k] = v
}

func (m *MissionPackage) AddFile(f FileContent) {
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
		buf.WriteString(fmt.Sprintf("<Content ignore=\"false\" zipEntry=\"%s\"/>", v.Name()))
	}

	buf.WriteString("</Contents>")

	return buf.Bytes()
}

func (m *MissionPackage) Create() ([]byte, error) {
	buff := new(bytes.Buffer)
	zipW := zip.NewWriter(buff)

	f, err := zipW.Create("MANIFEST/manifest.xml")
	if err != nil {
		return nil, err
	}

	_, err = f.Write(m.Manifest())

	if err != nil {
		return nil, err
	}

	for _, zf := range m.files {
		f1, err := zipW.Create(zf.Name())
		if err != nil {
			return nil, err
		}

		_, err = f1.Write(zf.Content())
		if err != nil {
			return nil, err
		}
	}

	err = zipW.Close()

	return buff.Bytes(), err
}

type FileContent interface {
	SetName(name string)
	Name() string
	Content() []byte
}

type FsFile struct {
	name string
	data []byte
}

func NewFsFile(name, path string) (*FsFile, error) {
	dat, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return &FsFile{name: name, data: dat}, nil
}

func (f *FsFile) Name() string {
	return f.name
}

func (f *FsFile) SetName(name string) {
	f.name = name
}

func (f *FsFile) Content() []byte {
	return f.data
}

type PrefFile struct {
	name string
	cls  string
	data map[string]any
}

func NewUserProfilePrefFile(prefix string) *PrefFile {
	return NewPrefFile(strings.TrimRight(prefix, "/")+"/user-profile.pref", "com.atakmap.app.civ_preferences")
}

func NewAppPrefFile() *PrefFile {
	return NewPrefFile("atak.pref", "com.atakmap.app_preferences")
}

func NewPrefFile(name, cls string) *PrefFile {
	return &PrefFile{name: name, cls: cls, data: make(map[string]any)}
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
	sb.WriteString("<preferences>")
	sb.WriteString(fmt.Sprintf("<preference version=\"1\" name=\"%s\">\n", p.cls))

	for k, v := range p.data {
		var cl string
		switch v.(type) {
		case bool:
			cl = "class java.lang.Boolean"
		default:
			cl = "class java.lang.String"
		}
		sb.WriteString(fmt.Sprintf("<entry key=\"%s\" class=\"%s\">%v</entry>\n", k, cl, v))
	}

	sb.WriteString("</preference></preferences>")

	return sb.Bytes()
}
