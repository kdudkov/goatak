package cot

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

const maxBufLen = 2048

type TagReader struct {
	r io.ByteReader
}

func NewTagReader(r io.Reader) *TagReader {
	if rb, ok := r.(io.ByteReader); ok {
		return &TagReader{r: rb}
	}

	return &TagReader{r: bufio.NewReader(r)}
}

func (er *TagReader) ReadTag() (string, []byte, error) {
	var buf bytes.Buffer

	var saved bytes.Buffer

	// start tag
	for {
		b, err := er.r.ReadByte()
		if err != nil {
			return "", nil, err
		}

		if b == '<' {
			break
		}
	}
	buf.WriteByte('<')

	for {
		b, err := er.r.ReadByte()
		if err != nil {
			return "", nil, err
		}

		buf.WriteByte(b)

		if b == '>' {
			break
		}

		if b == '<' {
			return "", nil, fmt.Errorf("bad xml: %s", buf.String())
		}

		if buf.Len() > maxBufLen {
			return "", nil, fmt.Errorf("too long tag")
		}
	}

	tag := buf.String()
	selfClosed := strings.HasSuffix(tag, "/>")

	if strings.HasPrefix(tag, "</") {
		return "", nil, fmt.Errorf("closed tag")
	}

	tag = strings.Trim(tag, "<>/")

	if strings.ContainsRune(tag, ' ') {
		tag = strings.SplitN(tag, " ", 2)[0]
	}

	if tag[0] == '?' {
		return tag, buf.Bytes(), nil
	}

	if selfClosed {
		return tag, buf.Bytes(), nil
	}

	saved.Write(buf.Bytes())
	// end tag
	buf.Reset()

	for {
		b, err := er.r.ReadByte()
		if err != nil {
			return tag, saved.Bytes(), err
		}

		saved.WriteByte(b)
		buf.WriteByte(b)

		if b == '<' {
			buf.Reset()
			buf.WriteByte('<')

			continue
		}

		if b == '>' {
			if buf.String() == "</"+tag+">" {
				return tag, saved.Bytes(), nil
			}

			buf.Reset()
		}

		if saved.Len() > maxBufLen {
			return "", nil, fmt.Errorf("too long tag")
		}
	}
}
