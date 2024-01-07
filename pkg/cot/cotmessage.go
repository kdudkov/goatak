package cot

import (
	"fmt"
	"strings"
	"time"

	"github.com/kdudkov/goatak/pkg/cotproto"
)

type CotMessage struct {
	From       string               `json:"from,omitempty"`
	Scope      string               `json:"scope"`
	TakMessage *cotproto.TakMessage `json:"tak_message"`
	Detail     *Node                `json:"-"`
}

func CotFromProto(msg *cotproto.TakMessage, from, scope string) (*CotMessage, error) {
	if msg == nil {
		return nil, nil
	}

	d, err := DetailsFromString(msg.GetCotEvent().GetDetail().GetXmlDetail())

	return &CotMessage{From: from, Scope: scope, TakMessage: msg, Detail: d}, err
}

func (m *CotMessage) GetSendTime() time.Time {
	if m == nil || m.TakMessage == nil {
		return time.Time{}
	}

	return TimeFromMillis(m.TakMessage.GetCotEvent().GetSendTime())
}

func (m *CotMessage) GetStartTime() time.Time {
	if m == nil || m.TakMessage == nil {
		return time.Time{}
	}

	return TimeFromMillis(m.TakMessage.GetCotEvent().GetStartTime())
}

func (m *CotMessage) GetStaleTime() time.Time {
	if m == nil || m.TakMessage == nil {
		return time.Time{}
	}

	return TimeFromMillis(m.TakMessage.GetCotEvent().GetStaleTime())
}

func (m *CotMessage) GetUID() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	return m.TakMessage.GetCotEvent().GetUid()
}

func (m *CotMessage) GetType() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	return m.TakMessage.GetCotEvent().GetType()
}

func (m *CotMessage) GetCallsign() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	if s := m.TakMessage.GetCotEvent().GetDetail().GetContact().GetCallsign(); s != "" {
		return s
	}

	// if phonenumber is in contact - contact is in xmldetails
	return m.GetDetail().GetFirst("contact").GetAttr("callsign")
}

func (m *CotMessage) GetEndpoint() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	if s := m.TakMessage.GetCotEvent().GetDetail().GetContact().GetEndpoint(); s != "" {
		return s
	}

	return m.GetDetail().GetFirst("contact").GetAttr("endpoint")
}

func (m *CotMessage) GetTeam() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	return m.TakMessage.GetCotEvent().GetDetail().GetGroup().GetName()
}

func (m *CotMessage) GetRole() string {
	if m == nil || m.TakMessage == nil {
		return ""
	}

	return m.TakMessage.GetCotEvent().GetDetail().GetGroup().GetRole()
}

func (m *CotMessage) GetStale() time.Time {
	if m == nil || m.TakMessage == nil {
		return time.Unix(0, 0)
	}

	return TimeFromMillis(m.TakMessage.GetCotEvent().GetStaleTime())
}

func (m *CotMessage) IsContact() bool {
	if m == nil || m.TakMessage == nil {
		return false
	}

	return strings.HasPrefix(m.GetType(), "a-f-") && m.GetEndpoint() != ""
}

func (m *CotMessage) IsChat() bool {
	if m == nil || m.TakMessage == nil {
		return false
	}

	return m.GetType() == "b-t-f"
}

func (m *CotMessage) GetDetail() *Node {
	if m == nil {
		return nil
	}

	return m.Detail
}

func (m *CotMessage) IsChatReceipt() bool {
	if m == nil || m.TakMessage == nil {
		return false
	}

	return m.GetType() == "b-t-f-r" || m.GetType() == "b-t-f-d"
}

func (m *CotMessage) PrintChat() string {
	chat := m.GetDetail().GetFirst("__chat")
	if chat == nil {
		return ""
	}

	from := chat.GetAttr("senderCallsign")
	to := chat.GetAttr("chatroom")
	text := m.GetDetail().GetFirst("remarks").GetText()

	return fmt.Sprintf("%s -> %s: \"%s\"", from, to, text)
}

func (m *CotMessage) GetLatLon() (float64, float64) {
	if m == nil {
		return 0, 0
	}

	return m.TakMessage.GetCotEvent().GetLat(), m.TakMessage.GetCotEvent().GetLon()
}

func (m *CotMessage) GetLat() float64 {
	if m == nil {
		return 0
	}

	return m.TakMessage.GetCotEvent().GetLat()
}

func (m *CotMessage) GetLon() float64 {
	if m == nil {
		return 0
	}

	return m.TakMessage.GetCotEvent().GetLon()
}

func (m *CotMessage) GetParent() (string, string) {
	if m.GetDetail() == nil {
		return "", ""
	}

	for _, link := range m.GetDetail().GetAll("link") {
		if link.GetAttr("relation") == "p-p" {
			return link.GetAttr("uid"), link.GetAttr("parent_callsign")
		}
	}

	return "", ""
}

func (m *CotMessage) GetIconsetPath() string {
	return m.GetDetail().GetFirst("usericon").GetAttr("iconsetpath")
}

func (m *CotMessage) GetColor() string {
	return m.GetDetail().GetFirst("color").GetAttr("argb")
}

func (m *CotMessage) GetFirstLink(relation string) *Node {
	if m.GetDetail() == nil {
		return nil
	}

	for _, link := range m.GetDetail().GetAll("link") {
		if link.GetAttr("relation") == relation {
			return link
		}
	}

	return nil
}

func TimeFromMillis(ms uint64) time.Time {
	return time.Unix(0, 1000000*int64(ms))
}

func TimeToMillis(t time.Time) uint64 {
	return uint64(t.UnixNano() / 1000000)
}
