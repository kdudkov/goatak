package cot

import (
	"cmp"
	"strings"
	"time"

	"github.com/kdudkov/goatak/pkg/cotproto"
)

const (
	LocalFrom      = "local"
	LocalScope     = "local"
	BroadcastScope = "broadcast"
)

type CotMessage struct {
	From       string               `json:"from,omitempty"`
	Scope      string               `json:"scope"`
	TakMessage *cotproto.TakMessage `json:"tak_message"`
	Detail     *Node                `json:"-"`
}

func LocalCotMessage(msg *cotproto.TakMessage) *CotMessage {
	m, _ := CotFromProto(msg, LocalFrom, LocalScope)

	return m
}

func CotFromProto(msg *cotproto.TakMessage, from, scope string) (*CotMessage, error) {
	if msg == nil {
		return nil, nil
	}

	d, err := DetailsFromString(msg.GetCotEvent().GetDetail().GetXmlDetail())

	return &CotMessage{From: from, Scope: scope, TakMessage: msg, Detail: d}, err
}

func (m *CotMessage) GetTakMessage() *cotproto.TakMessage {
	if m == nil {
		return nil
	}

	return m.TakMessage
}

func (m *CotMessage) GetUpdatedTakMessage() *cotproto.TakMessage {
	if m == nil {
		return nil
	}

	msg := m.GetTakMessage()

	if m.Detail != nil {
		if cot := msg.GetCotEvent(); cot != nil {
			if cot.GetDetail() == nil {
				cot.Detail = &cotproto.Detail{}
			}

			cot.Detail.XmlDetail = m.Detail.AsXMLString()
		}
	}

	return msg
}

func (m *CotMessage) GetSendTime() time.Time {
	if m == nil {
		return time.Time{}
	}

	return TimeFromMillis(m.GetTakMessage().GetCotEvent().GetSendTime())
}

func (m *CotMessage) GetStartTime() time.Time {
	if m == nil {
		return time.Time{}
	}

	return TimeFromMillis(m.GetTakMessage().GetCotEvent().GetStartTime())
}

func (m *CotMessage) GetStaleTime() time.Time {
	if m == nil {
		return time.Time{}
	}

	return TimeFromMillis(m.GetTakMessage().GetCotEvent().GetStaleTime())
}

func (m *CotMessage) GetUID() string {
	if m == nil {
		return ""
	}

	return m.GetTakMessage().GetCotEvent().GetUid()
}

func (m *CotMessage) GetType() string {
	if m == nil {
		return ""
	}

	return m.GetTakMessage().GetCotEvent().GetType()
}

func (m *CotMessage) GetCallsign() string {
	if m == nil {
		return ""
	}

	return cmp.Or(
		m.GetTakMessage().GetCotEvent().GetDetail().GetContact().GetCallsign(),
		m.GetDetail().GetFirst("contact").GetAttr("callsign"),
	)
}

func (m *CotMessage) GetEndpoint() string {
	if m == nil {
		return ""
	}

	return cmp.Or(
		m.GetTakMessage().GetCotEvent().GetDetail().GetContact().GetEndpoint(),
		m.GetDetail().GetFirst("contact").GetAttr("endpoint"),
	)
}

func (m *CotMessage) GetTakv() *cotproto.Takv {
	if m == nil {
		return nil
	}

	if s := m.GetTakMessage().GetCotEvent().GetDetail().GetTakv(); s != nil {
		return s
	}

	if c := m.Detail.GetFirst("takv"); c != nil {
		return &cotproto.Takv{
			Device:   c.GetAttr("device"),
			Platform: c.GetAttr("platform"),
			Os:       c.GetAttr("os"),
			Version:  c.GetAttr("version"),
		}
	}

	return nil
}

func (m *CotMessage) GetTeam() string {
	if m == nil {
		return ""
	}

	return cmp.Or(
		m.GetTakMessage().GetCotEvent().GetDetail().GetGroup().GetName(),
		m.GetDetail().GetFirst("__group").GetAttr("name"),
	)
}

func (m *CotMessage) GetRole() string {
	if m == nil {
		return ""
	}

	return cmp.Or(
		m.GetTakMessage().GetCotEvent().GetDetail().GetGroup().GetRole(),
		m.GetDetail().GetFirst("__group").GetAttr("role"),
	)
}

func (m *CotMessage) GetExRole() string {
	if m == nil {
		return ""
	}

	return m.GetDetail().GetFirst("__group").GetAttr("exrole")
}

func (m *CotMessage) IsContact() bool {
	if m == nil || m.GetTakMessage() == nil {
		return false
	}

	return strings.HasPrefix(m.GetType(), "a-f-") && m.GetEndpoint() != ""
}

func (m *CotMessage) IsChat() bool {
	if m == nil || m.GetTakMessage() == nil {
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
	if m == nil {
		return false
	}

	return m.GetType() == "b-t-f-d" || m.GetType() == "b-t-f-p" ||
		m.GetType() == "b-t-f-r" || m.GetType() == "b-t-f-s" || m.GetType() == "b-t-f-u"
}

func (m *CotMessage) IsFileTransfer() bool {
	if m == nil {
		return false
	}

	return m.GetType() == "b-f-t-r" || m.GetType() == "b-f-t-a"
}

func (m *CotMessage) IsPing() bool {
	return m.GetType() == "t-x-c-t" || m.GetType() == "t-x-c-t-r"
}

func (m *CotMessage) IsControl() bool {
	return MatchAnyPattern(m.GetType(),
		"t-b",
		"t-b-a",
		"t-b-c",
		"t-b-q",
		//"t-x-c-t",
		//"t-x-c-t-r",
		"t-x-takp-q",
		"t-x-c-m",
		"t-x-c-i-e",
		"t-x-c-i-d")
}

func (m *CotMessage) IsLocal() bool {
	return m.Scope == LocalScope
}

func (m *CotMessage) IsMapItem() bool {
	if MatchAnyPattern(m.GetType(), "a-", "u-") {
		return true
	}

	if m.IsChat() || m.IsChatReceipt() || m.IsFileTransfer() {
		return false
	}

	return (m.GetLat() != 0 && m.GetLon() != 0) && MatchAnyPattern(m.GetType(), "b-")
}

func (m *CotMessage) GetLatLon() (float64, float64) {
	if m == nil {
		return 0, 0
	}

	return m.GetTakMessage().GetCotEvent().GetLat(), m.GetTakMessage().GetCotEvent().GetLon()
}

func (m *CotMessage) GetLat() float64 {
	if m == nil {
		return 0
	}

	return m.GetTakMessage().GetCotEvent().GetLat()
}

func (m *CotMessage) GetLon() float64 {
	if m == nil {
		return 0
	}

	return m.GetTakMessage().GetCotEvent().GetLon()
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
		if relation == "" || link.GetAttr("relation") == relation {
			return link
		}
	}

	return nil
}

func TimeFromMillis(ms uint64) time.Time {
	return time.UnixMilli(int64(ms))
}

func TimeToMillis(t time.Time) uint64 {
	return uint64(t.UnixMilli())
}
