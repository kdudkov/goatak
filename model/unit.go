package model

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/kdudkov/goatak/cot"
)

const (
	staleContactDelete = time.Minute * 30
)

type Pos struct {
	time  time.Time
	lat   float64
	lon   float64
	speed float64
}

type Item struct {
	uid       string
	type_     string
	callsign  string
	staleTime time.Time
	startTime time.Time
	sendTime  time.Time
	received  time.Time
	msg       *cot.Msg
}

type Unit struct {
	Item
	parentCallsign string
	parentUid      string
	mx             sync.RWMutex
	track          []*Pos
}

type Contact struct {
	Item
	online   bool
	lastSeen time.Time
	mx       sync.RWMutex
	track    []*Pos
}

type Point struct {
	Item
	parentCallsign string
	parentUid      string
	color          int32
}

func (c *Contact) String() string {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return fmt.Sprintf("contact: %s %s %s", c.uid, c.type_, c.callsign)
}

func (u *Unit) String() string {
	u.mx.RLock()
	defer u.mx.RUnlock()
	return fmt.Sprintf("unit: %s %s %s", u.uid, u.type_, u.callsign)
}

func (p *Point) String() string {
	return fmt.Sprintf("point: %s %s %s", p.uid, p.type_, p.callsign)
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
	return u.staleTime.Before(time.Now())
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

func ContactFromMsg(msg *cot.Msg) *Contact {
	pos := &Pos{
		time:  cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetSendTime()),
		lat:   msg.TakMessage.GetCotEvent().GetLat(),
		lon:   msg.TakMessage.GetCotEvent().GetLon(),
		speed: msg.TakMessage.GetCotEvent().GetDetail().GetTrack().GetSpeed(),
	}

	return &Contact{
		Item:     ItemFromMsg(msg),
		online:   true,
		lastSeen: time.Now(),
		mx:       sync.RWMutex{},
		track:    []*Pos{pos},
	}
}

func UnitFromMsg(msg *cot.Msg) *Unit {
	pos := &Pos{
		time:  cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetSendTime()),
		lat:   msg.TakMessage.GetCotEvent().GetLat(),
		lon:   msg.TakMessage.GetCotEvent().GetLon(),
		speed: msg.TakMessage.GetCotEvent().GetDetail().GetTrack().GetSpeed(),
	}

	u := &Unit{
		Item:  ItemFromMsg(msg),
		mx:    sync.RWMutex{},
		track: []*Pos{pos},
	}

	u.parentUid, u.parentCallsign = msg.GetParent()

	return u
}

func PointFromEvent(msg *cot.Msg) *Point {
	p := &Point{
		Item: ItemFromMsg(msg),
	}

	p.parentUid, p.parentCallsign = msg.GetParent()

	if c := msg.Detail.GetFirstChild("color"); c != nil {
		if col, err := strconv.Atoi(c.GetAttr("argb")); err == nil {
			p.color = int32(col)
		}
	}

	return p
}

func ItemFromMsg(msg *cot.Msg) Item {
	return Item{
		uid:       msg.TakMessage.GetCotEvent().GetUid(),
		type_:     msg.TakMessage.GetCotEvent().GetType(),
		callsign:  msg.TakMessage.GetCotEvent().GetDetail().GetContact().GetCallsign(),
		staleTime: cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetStaleTime()),
		startTime: cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetStartTime()),
		sendTime:  cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetSendTime()),
		msg:       msg,
		received:  time.Now(),
	}
}

func (i *Item) GetLanLon() (float64, float64) {
	return i.msg.TakMessage.GetCotEvent().GetLat(), i.msg.TakMessage.GetCotEvent().GetLon()
}

func (c *Contact) Update(msg *cot.Msg) {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.online = true
	c.lastSeen = time.Now()
	if msg != nil {
		pos := getPos(c.msg, msg)
		c.msg = msg

		if pos != nil {
			c.track = append(c.track, pos)

			if len(c.track) > 5000 {
				c.track = c.track[len(c.track)-5000:]
			}
		}
	}
}

func (u *Unit) Update(msg *cot.Msg) {
	u.mx.Lock()
	defer u.mx.Unlock()

	u.Item = ItemFromMsg(msg)

	if msg != nil {
		pos := getPos(u.msg, msg)
		u.msg = msg

		link := msg.Detail.GetFirstChild("link")
		if link.GetAttr("relation") == "p-p" {
			u.parentCallsign = link.GetAttr("parent_callsign")
			u.parentUid = link.GetAttr("uid")
		}

		if pos != nil {
			u.track = append(u.track, pos)

			if len(u.track) > 5000 {
				u.track = u.track[len(u.track)-5000:]
			}
		}
	}
}

func (c *Contact) SetOffline() {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.online = false
}

func getPos(msg1, msg2 *cot.Msg) *Pos {
	if msg1 == nil || msg2 == nil {
		return nil
	}

	lat1, lon1 := msg1.GetLatLon()

	if lat1 == 0 && lon1 == 0 {
		return nil
	}

	lat2, lon2 := msg2.GetLatLon()

	if lat2 == 0 && lon2 == 0 {
		return nil
	}

	dist, _ := DistBea(lat1, lon1, lat2, lon2)

	if dist > 25 {
		return &Pos{
			time:  cot.TimeFromMillis(msg2.TakMessage.GetCotEvent().GetSendTime()),
			lat:   lat2,
			lon:   lon2,
			speed: msg2.TakMessage.GetCotEvent().GetDetail().GetTrack().GetSpeed(),
		}
	}

	return nil
}
