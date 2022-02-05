package model

import (
	"sync"
	"time"

	"github.com/kdudkov/goatak/cot"
	"github.com/kdudkov/goatak/cotproto"
)

const (
	staleContactDelete = time.Hour * 24
)

type Unit struct {
	uid      string
	type_    string
	callsign string
	stale    time.Time
	received time.Time
	msg      *cotproto.TakMessage
}

type Contact struct {
	uid      string
	type_    string
	callsign string
	stale    time.Time
	lastSeen time.Time
	msg      *cotproto.TakMessage
	online   bool
	mx       sync.RWMutex
}

type Point struct {
	uid      string
	type_    string
	name     string
	stale    time.Time
	received time.Time
	msg      *cotproto.TakMessage
}

func (u *Unit) GetMsg() *cotproto.TakMessage {
	return u.msg
}

func (c *Contact) GetMsg() *cotproto.TakMessage {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.msg
}

func (u *Unit) IsOld() bool {
	return u.stale.Before(time.Now())
}

func (c *Contact) IsOld() bool {
	c.mx.RLock()
	defer c.mx.RUnlock()

	return (!c.online) && c.lastSeen.Add(staleContactDelete).Before(time.Now())
}

func (c *Contact) GetUID() string {
	c.mx.RLock()
	defer c.mx.RUnlock()

	return c.uid
}

func (c *Contact) GetCallsign() string {
	c.mx.RLock()
	defer c.mx.RUnlock()

	return c.callsign
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

func ContactFromEvent(msg *cotproto.TakMessage) *Contact {
	return &Contact{
		uid:      msg.GetCotEvent().GetUid(),
		callsign: msg.GetCotEvent().GetDetail().GetContact().GetCallsign(),
		lastSeen: time.Now(),
		stale:    cot.TimeFromMillis(msg.GetCotEvent().GetStaleTime()),
		type_:    msg.GetCotEvent().GetType(),
		msg:      msg,
		online:   true,
		mx:       sync.RWMutex{},
	}
}

func UnitFromEvent(msg *cotproto.TakMessage) *Unit {
	return &Unit{
		uid:      msg.GetCotEvent().GetUid(),
		callsign: msg.GetCotEvent().GetDetail().GetContact().GetCallsign(),
		stale:    cot.TimeFromMillis(msg.GetCotEvent().GetStaleTime()),
		type_:    msg.GetCotEvent().GetType(),
		msg:      msg,
		received: time.Now(),
	}
}

func PointFromEvent(msg *cotproto.TakMessage) *Point {
	return &Point{
		uid:      msg.GetCotEvent().GetUid(),
		name:     msg.GetCotEvent().GetDetail().GetContact().GetCallsign(),
		stale:    cot.TimeFromMillis(msg.GetCotEvent().GetStaleTime()),
		type_:    msg.GetCotEvent().GetType(),
		msg:      msg,
		received: time.Now(),
	}
}

func (c *Contact) SetLastSeenNow(msg *cotproto.TakMessage) {
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
