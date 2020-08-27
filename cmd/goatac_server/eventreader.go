package main

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

type Eventreader struct {
	r io.ByteReader
}

func NewEventnReader(r io.Reader) *Eventreader {
	if rb, ok := r.(io.ByteReader); ok {
		return &Eventreader{r: rb}
	} else {
		return &Eventreader{r: bufio.NewReader(r)}
	}
}

func (er *Eventreader) ReadEvent() ([]byte, error) {
	var buf bytes.Buffer
	var saved bytes.Buffer

	// start tag
Loop:
	for {
		for {
			b, err := er.r.ReadByte()
			if err != nil {
				return nil, err
			}
			if b == '<' {
				break
			}
		}
		buf.WriteByte('<')

		for {
			b, err := er.r.ReadByte()
			if err != nil {
				return nil, err
			}
			buf.WriteByte(b)
			if b == '>' {
				break
			}
			if b == '<' {
				buf.Reset()
				buf.WriteByte(b)
			}
			if buf.Len() > 2048 {
				buf.Reset()
				continue Loop
			}
		}

		s := buf.String()
		if s == "<event>" || strings.HasPrefix(s, "<event ") {
			if strings.HasSuffix(s, "/>") {
				return buf.Bytes(), nil
			}
			saved.Write(buf.Bytes())
			break
		}
		buf.Reset()
	}

	// end tag
	buf.Reset()
	for {
		for {
			b, err := er.r.ReadByte()
			if err != nil {
				return nil, err
			}
			if b == '<' {
				break
			}
			saved.WriteByte(b)
		}
		buf.WriteByte('<')

		for {
			b, err := er.r.ReadByte()
			if err != nil {
				return nil, err
			}
			buf.WriteByte(b)
			if b == '>' {
				break
			}
		}

		if buf.String() == "</event>" {
			saved.Write(buf.Bytes())
			break
		}
		saved.Write(buf.Bytes())
		buf.Reset()
	}

	return saved.Bytes(), nil
}
