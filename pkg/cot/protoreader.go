package cot

import (
	"bufio"
	"encoding/binary"
	"google.golang.org/protobuf/proto"
	"io"

	"github.com/kdudkov/goatak/pkg/cotproto"
)

const magic byte = 0xbf

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

func (er *ProtoReader) ReadProtoBuf() (*cotproto.TakMessage, error) {
	return ReadProto(er.r)
}

func MakeProtoPacket(msg *cotproto.TakMessage) ([]byte, error) {
	buf1, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, len(buf1)+5)
	buf[0] = magic
	n := binary.PutUvarint(buf[1:], uint64(len(buf1)))
	copy(buf[1+n:], buf1)
	return buf[:1+n+len(buf1)+1], nil
}

func ReadProto(r *bufio.Reader) (*cotproto.TakMessage, error) {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == magic {
			break
		}
	}
	size, err := binary.ReadUvarint(r)

	if err != nil {
		return nil, err
	}

	buf := make([]byte, size)
	_, err = io.ReadFull(r, buf)

	if err != nil {
		return nil, err
	}

	msg := new(cotproto.TakMessage)
	err = proto.Unmarshal(buf, msg)
	return msg, err
}
