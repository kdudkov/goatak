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

type CotItem interface {
	GetCotType() string
}

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
	local     bool
	send      bool
}

type Unit struct {
	Item
	parentCallsign string
	parentUid      string
	mx             sync.RWMutex
	track          []*Pos
	color          uint32
	icon           string
}

type Contact struct {
	Item
	online bool
	mx     sync.RWMutex
	track  []*Pos
}

type Point struct {
	Item
	parentCallsign string
	parentUid      string
	color          uint32
	icon           string
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

func (u *Unit) GetCotType() string {
	return u.type_
}

func (c *Contact) GetCotType() string {
	return c.type_
}

func (p *Point) GetCotType() string {
	return p.type_
}

func (i *Item) GetMsg() *cot.Msg {
	return i.msg
}

func (c *Contact) GetMsg() *cot.Msg {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.msg
}

func (u *Unit) IsOld() bool {
	return u.staleTime.Before(time.Now())
}

func (p *Point) IsOld() bool {
	return p.staleTime.Before(time.Now())
}

func (c *Contact) IsOld() bool {
	c.mx.RLock()
	defer c.mx.RUnlock()

	return (!c.online) && c.received.Add(staleContactDelete).Before(time.Now())
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

func (c *Contact) GetReceived() time.Time {
	c.mx.RLock()
	defer c.mx.RUnlock()

	return c.received
}

func (c *Contact) GetStartTime() time.Time {
	c.mx.RLock()
	defer c.mx.RUnlock()

	return c.startTime
}

func (c *Contact) IsOnline() bool {
	c.mx.RLock()
	defer c.mx.RUnlock()

	return c.online
}

func (i *Item) IsLocal() bool {
	return i.local
}

func (i *Item) IsSend() bool {
	return i.send
}

func ContactFromMsg(msg *cot.Msg) *Contact {
	pos := &Pos{
		time:  cot.TimeFromMillis(msg.TakMessage.GetCotEvent().GetSendTime()),
		lat:   msg.TakMessage.GetCotEvent().GetLat(),
		lon:   msg.TakMessage.GetCotEvent().GetLon(),
		speed: msg.TakMessage.GetCotEvent().GetDetail().GetTrack().GetSpeed(),
	}

	return &Contact{
		Item:   ItemFromMsg(msg),
		online: true,
		mx:     sync.RWMutex{},
		track:  []*Pos{pos},
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
	if c := msg.Detail.GetFirstChild("color"); c != nil {
		if col, err := strconv.Atoi(c.GetAttr("argb")); err == nil {
			u.color = uint32(col)
		}
	}
	u.icon = msg.Detail.GetFirstChild("usericon").GetAttr("iconsetpath")

	return u
}

func UnitFromMsgLocal(msg *cot.Msg, local, send bool) *Unit {
	u := UnitFromMsg(msg)
	u.local = local
	u.send = send
	return u
}

func PointFromMsg(msg *cot.Msg) *Point {
	p := &Point{
		Item: ItemFromMsg(msg),
	}

	p.parentUid, p.parentCallsign = msg.GetParent()

	if c := msg.Detail.GetFirstChild("color"); c != nil {
		if col, err := strconv.Atoi(c.GetAttr("argb")); err == nil {
			p.color = uint32(col)
		}
	}
	p.icon = msg.Detail.GetFirstChild("usericon").GetAttr("iconsetpath")

	return p
}

func PointFromMsgLocal(msg *cot.Msg, local, send bool) *Point {
	p := PointFromMsg(msg)
	p.local = local
	p.send = send
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
	c.received = time.Now()
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
