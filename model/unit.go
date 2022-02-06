package model

import (
	"sync"
	"time"

	"github.com/kdudkov/goatak/cot"
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
	msg      *cot.Msg
}

type Contact struct {
	uid      string
	type_    string
	callsign string
	stale    time.Time
	lastSeen time.Time
	msg      *cot.Msg
	online   bool
	mx       sync.RWMutex
}

type Point struct {
	uid      string
	type_    string
	name     string
	stale    time.Time
	received time.Time
	msg      *cot.Msg
}

func (u *Unit) GetMsg() *cot.Msg {
	return u.msg
}

func (c *Contact) GetMsg() *cot.Msg {
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

func ContactFromEvent(msg *cot.Msg) *Contact {
	return &Contact{
		uid:      msg.TakMessage.GetCotEvent().GetUid(),
		callsign: msg.TakMessage.GetCotEvent().GetDetail().GetContact().GetCallsign(),
		lastSeen: time.Now(),
		stale:    cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetStaleTime()),
		type_:    msg.TakMessage.GetCotEvent().GetType(),
		msg:      msg,
		online:   true,
		mx:       sync.RWMutex{},
	}
}

func UnitFromEvent(msg *cot.Msg) *Unit {
	return &Unit{
		uid:      msg.TakMessage.GetCotEvent().GetUid(),
		callsign: msg.TakMessage.GetCotEvent().GetDetail().GetContact().GetCallsign(),
		stale:    cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetStaleTime()),
		type_:    msg.TakMessage.GetCotEvent().GetType(),
		msg:      msg,
		received: time.Now(),
	}
}

func PointFromEvent(msg *cot.Msg) *Point {
	return &Point{
		uid:      msg.TakMessage.GetCotEvent().GetUid(),
		name:     msg.TakMessage.GetCotEvent().GetDetail().GetContact().GetCallsign(),
		stale:    cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetStaleTime()),
		type_:    msg.TakMessage.GetCotEvent().GetType(),
		msg:      msg,
		received: time.Now(),
	}
}

func (c *Contact) SetLastSeenNow(msg *cot.Msg) {
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
