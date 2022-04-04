package cot

import (
	"fmt"
	"strings"
	"time"

	"github.com/kdudkov/goatak/cotproto"
)

type Msg struct {
	From       string
	TakMessage *cotproto.TakMessage
	Detail     *XMLDetails
}

func (m *Msg) GetUID() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	return m.TakMessage.GetCotEvent().UID
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

func (m *Msg) GetCallsignTo() string {
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

	return m.GetType() == "b-t-f" && m.Detail != nil && m.Detail.HasChild("__chat")
}

func (m *Msg) PrintChat() string {
	chat := m.Detail.GetFirstChild("__chat")
	if chat == nil {
		return ""
	}

	from := chat.GetAttr("senderCallsign")
	to := chat.GetAttr("chatroom")
	text, _ := m.Detail.GetChildValue("remarks")

	return fmt.Sprintf("%s -> %s: \"%s\"", from, to, text)
}

func (m *Msg) GetLatLon() (float64, float64) {
	if m == nil {
		return 0, 0
	}

	return m.TakMessage.GetCotEvent().GetLat(), m.TakMessage.GetCotEvent().GetLon()
}

func (m *Msg) GetLat() float64 {
	if m == nil {
		return 0
	}

	return m.TakMessage.GetCotEvent().GetLat()
}

func (m *Msg) GetLon() float64 {
	if m == nil {
		return 0
	}

	return m.TakMessage.GetCotEvent().GetLon()
}

func (m *Msg) GetParent() (string, string) {
	link := m.Detail.GetFirstChild("link")
	if link.GetAttr("relation") == "p-p" {
		return link.GetAttr("uid"), link.GetAttr("parent_callsign")
	}
	return "", ""
}

func TimeFromMillis(ms uint64) time.Time {
	return time.Unix(0, 1000000*int64(ms))
}

func TimeToMillis(t time.Time) uint64 {
	return uint64(t.UnixNano() / 1000000)
}
