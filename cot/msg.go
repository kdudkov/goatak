package cot

import (
	"strings"
	"time"

	"github.com/kdudkov/goatak/cotproto"
	"github.com/kdudkov/goatak/cotxml"
)

type Msg struct {
	TakMessage *cotproto.TakMessage
	Detail     *cotxml.XMLDetail
}

func (m *Msg) GetUid() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	return m.TakMessage.GetCotEvent().Uid
}

func (m *Msg) GetType() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	return m.TakMessage.GetCotEvent().Type
}

func (m *Msg) GetCallsign() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	return m.TakMessage.GetCotEvent().GetDetail().GetContact().GetCallsign()
}

func (m *Msg) GetStale() time.Time {
	if m == nil || m.TakMessage == nil {
		return time.Unix(0, 0)
	}

	return TimeFromMillis(m.TakMessage.CotEvent.StaleTime)
}

func (m *Msg) IsContact() bool {
	if m == nil || m.TakMessage == nil {
		return false
	}

	return strings.HasPrefix(m.GetType(), "a-f-") && m.TakMessage.GetCotEvent().GetDetail().GetContact().GetEndpoint() != ""
}

func (m *Msg) IsChat() bool {
	if m == nil || m.TakMessage == nil {
		return false
	}

	return m.GetType() == "b-t-f" && m.Detail != nil && m.Detail.Chat != nil
}

func TimeFromMillis(ms uint64) time.Time {
	return time.Unix(0, 1000000*int64(ms))
}

func TimeToMillis(t time.Time) uint64 {
	return uint64(t.UnixNano() / 1000000)
}
