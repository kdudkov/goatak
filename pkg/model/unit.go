package model

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kdudkov/goatak/pkg/cot"
)

const (
	StaleContactDelete = time.Hour * 12
	POINT              = "point"
	UNIT               = "unit"
	CONTACT            = "contact"
	MaxTrackPoints     = 5000
)

type Item struct {
	mx       sync.RWMutex
	uid      string
	class    string
	online   bool
	lastSeen time.Time
	local    bool
	send     bool
	track    []*Pos
	msg      *cot.CotMessage
}

func (i *Item) String() string {
	i.mx.RLock()
	defer i.mx.RUnlock()

	return fmt.Sprintf("%s: %s %s %s", i.class, i.uid, i.msg.GetType(), i.msg.GetCallsign())
}

func (i *Item) GetClass() string {
	i.mx.RLock()
	defer i.mx.RUnlock()

	return i.class
}

func (i *Item) GetType() string {
	i.mx.RLock()
	defer i.mx.RUnlock()

	return i.msg.GetType()
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

	return i.msg.GetCallsign()
}

func (i *Item) GetLastSeen() time.Time {
	i.mx.RLock()
	defer i.mx.RUnlock()

	return i.lastSeen
}

func (i *Item) IsOld() bool {
	i.mx.RLock()
	defer i.mx.RUnlock()

	switch i.class {
	case CONTACT:
		return (!i.online) && time.Since(i.lastSeen) > StaleContactDelete
	default:
		return i.msg.GetStaleTime().Before(time.Now())
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

func (i *Item) SetLocal(local bool) {
	i.mx.Lock()
	defer i.mx.Unlock()
	i.local = local
}

func (i *Item) SetSend(send bool) {
	i.mx.Lock()
	defer i.mx.Unlock()
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
		mx:       sync.RWMutex{},
		uid:      msg.GetUID(),
		class:    cls,
		lastSeen: time.Now(),
		online:   true,
		local:    false,
		send:     false,
		track:    nil,
		msg:      msg,
	}

	if i.class == UNIT || i.class == CONTACT {
		if msg.GetLat() != 0 || msg.GetLon() != 0 {
			pos := &Pos{
				time:  msg.GetSendTime(),
				lat:   msg.GetLat(),
				lon:   msg.GetLon(),
				speed: msg.TakMessage.GetCotEvent().GetDetail().GetTrack().GetSpeed(),
				mx:    sync.RWMutex{},
			}

			i.track = []*Pos{pos}
		}
	}

	return i
}

func (i *Item) GetLanLon() (float64, float64) {
	return i.msg.GetLat(), i.msg.GetLon()
}

func (i *Item) Update(msg *cot.CotMessage) {
	if msg == nil {
		i.SetOnline()

		return
	}

	i.mx.Lock()
	defer i.mx.Unlock()

	i.lastSeen = time.Now()
	i.class = GetClass(msg)
	i.msg = msg

	if i.class == UNIT || i.class == CONTACT {
		i.online = true

		if msg.GetLat() != 0 || msg.GetLon() != 0 {
			pos := &Pos{
				time:  msg.GetSendTime(),
				lat:   msg.GetLat(),
				lon:   msg.GetLon(),
				speed: msg.TakMessage.GetCotEvent().GetDetail().GetTrack().GetSpeed(),
				ce:    msg.TakMessage.GetCotEvent().GetCe(),
				mx:    sync.RWMutex{},
			}

			i.track = append(i.track, pos)

			if len(i.track) > MaxTrackPoints {
				i.track = i.track[len(i.track)-MaxTrackPoints:]
			}
		}
	}
}

func (i *Item) GetTrack() []*Pos {
	i.mx.RLock()
	defer i.mx.RUnlock()

	return i.track
}
