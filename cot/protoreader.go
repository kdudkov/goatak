package cot

import (
	"bufio"
	"encoding/binary"
	"io"
)

type ProtoReader struct {
	r *bufio.Reader
}

func NewProtoReader(r io.Reader) *ProtoReader {
	if rb, ok := r.(*bufio.Reader); ok {
		return &ProtoReader{r: rb}
	} else {
		return &ProtoReader{r: bufio.NewReader(r)}
	}
}

func (er *ProtoReader) ReadProtoBuf() ([]byte, error) {
	// start magic number
	for {
		b, err := er.r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == 0xbf {
			break
		}
	}
	size, err := binary.ReadUvarint(er.r)

	if err != nil {
		return nil, err
	}

	buf := make([]byte, size)
	_, err = io.ReadFull(er.r, buf)

	return buf, err
}
