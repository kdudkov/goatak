package model

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

type Point struct {
	ID          uint   `gorm:"primaryKey"`
	Scope       string `gorm:"index;size:255"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	UID         string `gorm:"uniqueIndex;size:255"`
	CreatorUID  string `gorm:"size:255"`
	Type        string `gorm:"size:255"`
	Callsign    string `gorm:"size:255"`
	Title       string `gorm:"size:255"`
	IconsetPath string `gorm:"size:255"`
	Color       string `gorm:"size:255"`
	Lat         float64
	Lon         float64
	EventData   []byte
	event       *cotproto.CotEvent `gorm:"-"`
}

func (p *Point) String() string {
	if p == nil {
		return "nil"
	}

	return fmt.Sprintf("%s %s %s", p.UID, p.Callsign, p.Title)
}

func (p *Point) GetEvent() *cotproto.CotEvent {
	if p == nil {
		return nil
	}

	if p.event != nil {
		return p.event
	}

	p.event = new(cotproto.CotEvent)

	if len(p.EventData) > 0 {
		_ = proto.Unmarshal(p.EventData, p.event)
	}

	return p.event
}

func (p *Point) BeforeSave(_ *gorm.DB) error {
	if p == nil {
		return nil
	}

	if p.UID == "" {
		return fmt.Errorf("empty uid")
	}

	if p.event == nil {
		p.EventData = nil
		return nil
	}

	var err error

	p.EventData, err = proto.Marshal(p.event)
	return err
}

func (p *Point) UpdateFromMsg(msg *cot.CotMessage) {
	parent, _ := msg.GetParent()
	p.CreatorUID = parent
	p.Scope = msg.Scope
	p.CreatedAt = msg.GetStartTime()
	p.Type = msg.GetType()
	p.Callsign = msg.GetCallsign()
	p.IconsetPath = msg.GetIconsetPath()
	p.Color = msg.GetColor()
	p.Lat = msg.GetLat()
	p.Lon = msg.GetLon()
	p.event = msg.GetTakMessage().GetCotEvent()

	return
}
