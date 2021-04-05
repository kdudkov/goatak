package model

import (
	"sync"
	"time"

	v1 "github.com/kdudkov/goatak/cot/v1"
)

const (
	staleContactDelete = time.Hour * 24
)

type Unit struct {
	Uid      string
	Type     string
	Callsign string
	Stale    time.Time
	Received time.Time
	msg      *v1.TakMessage
}

type Contact struct {
	uid      string
	type_    string
	callsign string
	stale    time.Time
	lastSeen time.Time
	msg      *v1.TakMessage
	online   bool
	mx       sync.RWMutex
}

func (u *Unit) IsOld() bool {
	return u.Stale.Before(time.Now())
}

func (c *Contact) IsOld() bool {
	c.mx.RLock()
	defer c.mx.RUnlock()

	return (!c.online) && c.lastSeen.Add(staleContactDelete).Before(time.Now())
}

func (c *Contact) GetLastSeen() time.Time {
	c.mx.RLock()
	defer c.mx.RUnlock()

	return c.lastSeen
}

func (c *Contact) IsOnline() bool {
	c.mx.RLock()
	defer c.mx.RUnlock()

	return c.online
}

func ContactFromEvent(msg *v1.TakMessage) *Contact {
	return &Contact{
		uid:      msg.GetCotEvent().GetUid(),
		callsign: msg.GetCotEvent().GetDetail().GetContact().GetCallsign(),
		lastSeen: time.Now(),
		stale:    v1.TimeFromMillis(msg.GetCotEvent().GetStaleTime()),
		type_:    msg.GetCotEvent().GetType(),
		msg:      msg,
		online:   true,
		mx:       sync.RWMutex{},
	}
}

func UnitFromEvent(msg *v1.TakMessage) *Unit {
	return &Unit{
		Uid:      msg.GetCotEvent().GetUid(),
		Callsign: msg.GetCotEvent().GetDetail().GetContact().GetCallsign(),
		Stale:    v1.TimeFromMillis(msg.GetCotEvent().GetStaleTime()),
		Type:     msg.GetCotEvent().GetType(),
		msg:      msg,
		Received: time.Now(),
	}
}

func (c *Contact) SetLastSeenNow(msg *v1.TakMessage) {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.online = true
	c.lastSeen = time.Now()
	if msg != nil {
		c.msg = msg
	}
}

func (c *Contact) SetOffline() {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.online = false
}
