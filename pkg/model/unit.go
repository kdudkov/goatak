package model

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kdudkov/goatak/pkg/cot"
)

const (
	staleContactDelete = time.Hour * 12
	POINT              = "point"
	UNIT               = "unit"
	CONTACT            = "contact"
	MAX_TRACK_POINTS   = 5000
)

type Pos struct {
	time  time.Time
	lat   float64
	lon   float64
	speed float64
}

type Item struct {
	mx             sync.RWMutex
	uid            string
	cottype        string
	class          string
	callsign       string
	staleTime      time.Time
	startTime      time.Time
	sendTime       time.Time
	lastSeen       time.Time
	online         bool
	local          bool
	send           bool
	parentCallsign string
	parentUid      string
	color          uint32
	icon           string
	track          []*Pos
	msg            *cot.CotMessage
}

func (i *Item) String() string {
	i.mx.RLock()
	defer i.mx.RUnlock()
	return fmt.Sprintf("%s: %s %s %s", i.class, i.uid, i.cottype, i.callsign)
}

func (i *Item) GetClass() string {
	i.mx.RLock()
	defer i.mx.RUnlock()
	return i.class
}

func (i *Item) GetCotType() string {
	i.mx.RLock()
	defer i.mx.RUnlock()
	return i.cottype
}

func (i *Item) GetMsg() *cot.CotMessage {
	i.mx.RLock()
	defer i.mx.RUnlock()
	return i.msg
}

func (i *Item) GetUID() string {
	i.mx.RLock()
	defer i.mx.RUnlock()
	return i.uid
}

func (i *Item) GetCallsign() string {
	i.mx.RLock()
	defer i.mx.RUnlock()
	return i.callsign
}

func (i *Item) GetLastSeen() time.Time {
	i.mx.RLock()
	defer i.mx.RUnlock()
	return i.lastSeen
}

func (i *Item) GetStartTime() time.Time {
	i.mx.RLock()
	defer i.mx.RUnlock()
	return i.startTime
}

func (i *Item) IsOld() bool {
	i.mx.RLock()
	defer i.mx.RUnlock()

	switch i.class {
	case CONTACT:
		return (!i.online) && i.lastSeen.Add(staleContactDelete).Before(time.Now())
	default:
		return i.staleTime.Before(time.Now())
	}
}

func (i *Item) IsOnline() bool {
	i.mx.RLock()
	defer i.mx.RUnlock()
	return i.online
}

func (i *Item) SetOffline() {
	i.mx.Lock()
	defer i.mx.Unlock()
	i.online = false
}

func (i *Item) SetOnline() {
	i.mx.Lock()
	defer i.mx.Unlock()
	i.online = true
	i.lastSeen = time.Now()
}

func (i *Item) SetLocal(local, send bool) {
	i.mx.Lock()
	defer i.mx.Unlock()
	i.local = local
	i.send = send
}

func (i *Item) IsSend() bool {
	i.mx.RLock()
	defer i.mx.RUnlock()
	return i.send
}

func GetClass(msg *cot.CotMessage) string {
	if msg == nil {
		return ""
	}
	t := msg.GetType()

	switch {
	case strings.HasPrefix(t, "a-"):
		if msg.IsContact() {
			return CONTACT
		} else {
			return UNIT
		}
	case strings.HasPrefix(t, "b-"):
		return POINT
	}

	return ""
}

func FromMsg(msg *cot.CotMessage) *Item {
	cls := GetClass(msg)

	if cls == "" {
		return nil
	}

	i := &Item{
		class:     cls,
		uid:       msg.GetUid(),
		cottype:   msg.GetType(),
		callsign:  msg.GetCallsign(),
		staleTime: cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetStaleTime()),
		startTime: cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetStartTime()),
		sendTime:  cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetSendTime()),
		msg:       msg,
		lastSeen:  time.Now(),
		online:    true,
		mx:        sync.RWMutex{},
	}

	i.parentUid, i.parentCallsign = msg.GetParent()

	if c := msg.Detail.GetFirst("color"); c != nil {
		if col, err := strconv.Atoi(c.GetAttr("argb")); err == nil {
			i.color = uint32(col)
		}
	}

	i.icon = msg.Detail.GetFirst("usericon").GetAttr("iconsetpath")

	if i.class == UNIT || i.class == CONTACT {
		if msg.TakMessage.GetCotEvent().GetLat() != 0 || msg.TakMessage.GetCotEvent().GetLat() != 0 {
			pos := &Pos{
				time:  cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetSendTime()),
				lat:   msg.TakMessage.GetCotEvent().GetLat(),
				lon:   msg.TakMessage.GetCotEvent().GetLon(),
				speed: msg.TakMessage.GetCotEvent().GetDetail().GetTrack().GetSpeed(),
			}

			i.track = []*Pos{pos}
		}
	}
	return i
}

func FromMsgLocal(msg *cot.CotMessage, send bool) *Item {
	i := FromMsg(msg)
	i.local = true
	i.send = send
	return i
}

func (i *Item) GetLanLon() (float64, float64) {
	return i.msg.TakMessage.GetCotEvent().GetLat(), i.msg.TakMessage.GetCotEvent().GetLon()
}

func (i *Item) Update(msg *cot.CotMessage) {
	if msg == nil {
		i.SetOnline()
		return
	}

	i.mx.Lock()
	defer i.mx.Unlock()

	i.class = GetClass(msg)
	i.cottype = msg.TakMessage.GetCotEvent().GetType()
	i.callsign = msg.TakMessage.GetCotEvent().GetDetail().GetContact().GetCallsign()
	i.staleTime = cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetStaleTime())
	i.startTime = cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetStartTime())
	i.sendTime = cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetSendTime())
	i.msg = msg
	i.lastSeen = time.Now()

	i.parentUid, i.parentCallsign = msg.GetParent()

	if c := msg.Detail.GetFirst("color"); c != nil {
		if col, err := strconv.Atoi(c.GetAttr("argb")); err == nil {
			i.color = uint32(col)
		}
	}

	i.icon = msg.Detail.GetFirst("usericon").GetAttr("iconsetpath")

	if i.class == UNIT || i.class == CONTACT {
		i.online = true

		if msg.TakMessage.GetCotEvent().GetLat() != 0 || msg.TakMessage.GetCotEvent().GetLat() != 0 {
			pos := &Pos{
				time:  cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetSendTime()),
				lat:   msg.TakMessage.GetCotEvent().GetLat(),
				lon:   msg.TakMessage.GetCotEvent().GetLon(),
				speed: msg.TakMessage.GetCotEvent().GetDetail().GetTrack().GetSpeed(),
			}

			i.track = append(i.track, pos)
			if len(i.track) > MAX_TRACK_POINTS {
				i.track = i.track[len(i.track)-MAX_TRACK_POINTS:]
			}
		}
	}
}

func (i *Item) UpdateFromWeb(w *WebUnit, m *cot.CotMessage) {
	if w == nil {
		return
	}

	i.mx.Lock()
	defer i.mx.Unlock()

	i.class = w.Category
	i.cottype = w.Type
	i.callsign = w.Callsign
	i.staleTime = w.StaleTime
	i.startTime = w.StartTime
	i.sendTime = w.SendTime
	i.lastSeen = time.Now()
	i.parentUid = w.ParentUid
	i.parentCallsign = w.ParentCallsign
	i.icon = w.Icon
	i.local = w.Local
	i.send = w.Send

	if w.Color != "" {
		if col, err := strconv.Atoi(w.Color); err == nil {
			i.color = uint32(col)
		}
	}

	i.msg = m
}
