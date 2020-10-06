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
	Uid      string
	Type     string
	Callsign string
	Stale    time.Time
	Received time.Time
	Evt      *cot.Event
}

type Contact struct {
	uid      string
	type_    string
	callsign string
	stale    time.Time
	lastSeen time.Time
	evt      *cot.Event
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

func ContactFromEvent(evt *cot.Event) *Contact {
	return &Contact{
		uid:      evt.Uid,
		callsign: evt.GetCallsign(),
		lastSeen: time.Now(),
		stale:    evt.Stale,
		type_:    evt.Type,
		evt:      evt,
		online:   true,
		mx:       sync.RWMutex{},
	}
}

func UnitFromEvent(evt *cot.Event) *Unit {
	return &Unit{
		Uid:      evt.Uid,
		Callsign: evt.GetCallsign(),
		Stale:    evt.Stale,
		Type:     evt.Type,
		Evt:      evt,
		Received: time.Now(),
	}
}

func (c *Contact) SetLastSeenNow(evt *cot.Event) {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.online = true
	c.lastSeen = time.Now()
	if evt != nil {
		c.evt = evt
	}
}

func (c *Contact) SetOffline() {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.online = false
}
