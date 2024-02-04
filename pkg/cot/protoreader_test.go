package cot

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtoRW(t *testing.T) {
	msg := MakeDpMsg("testuid", "a-f-G", "test", 10, 20)

	b, err := MakeProtoPacket(msg)
	require.NoError(t, err)

	msg1, err := ReadProto(bufio.NewReader(bytes.NewReader(b)))
	require.NoError(t, err)

	assert.Equal(t, "testuid.SPI1", msg1.GetCotEvent().GetUid())
	assert.Equal(t, "b-m-p-s-p-i", msg1.GetCotEvent().GetType())
	assert.InDelta(t, 10., msg1.GetCotEvent().GetLat(), 0.0001)
	assert.InDelta(t, 20., msg1.GetCotEvent().GetLon(), 0.0001)
}
