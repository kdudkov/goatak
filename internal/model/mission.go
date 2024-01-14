package model

import (
	"time"

	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/cotproto"
)

type Mission struct {
	ID             uint   `gorm:"primarykey"`
	Name           string `gorm:"index"`
	Username       string
	CreatorUID     string
	CreateTime     time.Time
	LastEdit       time.Time
	BaseLayer      string
	Bbox           string
	ChatRoom       string
	Classification string
	Description    string
	InviteOnly     bool
	Password       string
	Path           string
	Tool           string
	Groups         string
	Keywords       string
	Items          []*DataItem
}

type Subscription struct {
	ID          uint   `gorm:"primarykey"`
	MissionName string `gorm:"index"`
	ClientUID   string `gorm:"index"`
	Username    string
	CreateTime  time.Time
	Role        string
}

type Invitation struct {
	ID          uint   `gorm:"primarykey"`
	MissionName string `gorm:"index"`
	Typ         string `gorm:"index"`
	Invitee     string `gorm:"index"`
	CreatorUID  string
	CreateTime  time.Time
	Role        string
}

type DataItem struct {
	ID          uint `gorm:"primarykey"`
	MissionID   uint
	UID         string `gorm:"index"`
	CreatorUID  string
	Timestamp   time.Time
	Type        string
	Callsign    string
	Title       string
	IconsetPath string
	Color       string
	Lat         float64
	Lon         float64
	EventData   []byte
	Event       *cotproto.CotEvent `gorm:"-"`
}

func (d *DataItem) AfterFind(tx *gorm.DB) error {
	if d == nil || len(d.EventData) == 0 {
		return nil
	}

	d.Event = new(cotproto.CotEvent)

	return proto.Unmarshal(d.EventData, d.Event)
}

func (d *DataItem) BeforeUpdate(tx *gorm.DB) error {
	if d == nil {
		return nil
	}

	if d.Event == nil {
		d.EventData = nil
		return nil
	}

	var err error

	d.EventData, err = proto.Marshal(d.Event)
	return err
}
