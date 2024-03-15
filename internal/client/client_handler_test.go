package client

import (
	"bytes"
	"encoding/binary"
	"github.com/spf13/viper"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

func TestRoute(t *testing.T) {
	h := NewConnClientHandler("test", nil, &HandlerConfig{UID: "111", IsClient: true})
	h.ver = 1
	h.user = &model.User{Scope: "aaa", ReadScope: []string{"ccc", "ddd"}}

	var msg *cot.CotMessage

	var c *cotproto.TakMessage

	var err error

	msg = &cot.CotMessage{TakMessage: cot.MakePing("123"), Scope: "aaa"}
	c, err = passMsg(h, msg)
	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "t-x-c-t", c.GetCotEvent().GetType())

	msg = &cot.CotMessage{TakMessage: cot.MakePing("123"), Scope: "ddd"}
	c, err = passMsg(h, msg)
	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "t-x-c-t", c.GetCotEvent().GetType())

	msg = &cot.CotMessage{TakMessage: cot.MakePing("123"), Scope: "bbb"}
	c, err = passMsg(h, msg)
	require.NoError(t, err)
	assert.Nil(t, c)
}

func TestRouteChat(t *testing.T) {
	h := NewConnClientHandler("test", nil, &HandlerConfig{UID: "111", IsClient: true})
	h.ver = 1
	h.user = &model.User{Scope: "aaa"}

	var msg *cot.CotMessage

	var c *cotproto.TakMessage

	var err error

	tak := cot.BasicMsg("b-t-f", "123", time.Second*10)
	tak.CotEvent.Lat = 10.
	tak.CotEvent.Lon = 20.

	msg = &cot.CotMessage{TakMessage: tak, Scope: "aaa"}
	c, err = passMsg(h, msg)
	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "b-t-f", c.GetCotEvent().GetType())
	assert.InDelta(t, 10., c.GetCotEvent().GetLat(), 0.0001)
	assert.InDelta(t, 20., c.GetCotEvent().GetLon(), 0.0001)

	msg = &cot.CotMessage{TakMessage: tak, Scope: "bbb"}
	viper.Set("interscope_chat", false)
	c, err = passMsg(h, msg)
	require.NoError(t, err)
	assert.Nil(t, c)

	viper.Set("interscope_chat", true)
	c, err = passMsg(h, msg)
	require.NoError(t, err)
	assert.Equal(t, "b-t-f", c.GetCotEvent().GetType())
	assert.InDelta(t, 0., c.GetCotEvent().GetLat(), 0.0001)
	assert.InDelta(t, 0., c.GetCotEvent().GetLon(), 0.0001)
}

func passMsg(h *ConnClientHandler, msg *cot.CotMessage) (*cotproto.TakMessage, error) {
	if err := h.SendMsg(msg); err != nil {
		return nil, err
	}

	select {
	case dat := <-h.sendChan:
		bb := bytes.NewBuffer(dat)

		_, err := bb.ReadByte()
		if err != nil {
			return nil, err
		}

		size, err := binary.ReadUvarint(bb)
		if err != nil {
			return nil, err
		}

		buf := make([]byte, size)
		_, err = io.ReadFull(bb, buf)

		if err != nil {
			return nil, err
		}

		msg := new(cotproto.TakMessage)
		err = proto.Unmarshal(buf, msg)

		return msg, err
	default:
		return nil, nil
	}
}
