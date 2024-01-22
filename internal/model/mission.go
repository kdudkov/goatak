package model

import (
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/cotproto"
)

const sep = ","

type Mission struct {
	ID             uint   `gorm:"primarykey"`
	Scope          string `gorm:"index"`
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
	Hashes         string
	Items          []*DataItem
}

func (m *Mission) GetHashes() []string {
	if m.Hashes == "" {
		return nil
	}

	return strings.Split(m.Hashes, sep)
}

func (m *Mission) AddHashes(hashes ...string) bool {
	oldHashes := m.GetHashes()

	added := false

	for _, hash := range hashes {
		h := strings.Trim(hash, " \t\r\n")
		if hasItem(oldHashes, h) {
			continue
		}

		oldHashes = append(oldHashes, h)
		added = true
	}

	if added {
		m.Hashes = strings.Join(oldHashes, sep)

		return true
	}

	return false
}

func (m *Mission) RemoveHash(hash string) bool {
	hashes := m.GetHashes()
	newHashes := make([]string, 0)

	for _, h := range hashes {
		if h != hash && h != "" {
			newHashes = append(newHashes, h)
		}
	}

	if len(newHashes) != len(hashes) {
		m.Hashes = strings.Join(newHashes, sep)
		return true
	}

	return false
}

type Subscription struct {
	ID         uint   `gorm:"primarykey"`
	MissionID  uint   `gorm:"index"`
	ClientUID  string `gorm:"index"`
	Username   string
	CreateTime time.Time
	Role       string
}

type Invitation struct {
	ID         uint   `gorm:"primarykey"`
	MissionID  uint   `gorm:"index"`
	Typ        string `gorm:"index"`
	Invitee    string `gorm:"index"`
	CreatorUID string
	CreateTime time.Time
	Role       string
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

func (d *DataItem) UpdateFromMsg(msg *cot.CotMessage) {
	p, _ := msg.GetParent()
	d.CreatorUID = p
	d.Timestamp = msg.GetStartTime()
	d.Type = msg.GetType()
	d.Callsign = msg.GetCallsign()
	d.IconsetPath = msg.GetIconsetPath()
	d.Color = msg.GetColor()
	d.Lat = msg.GetLat()
	d.Lon = msg.GetLon()

	return
}

type Change struct {
	ID          uint      `gorm:"primarykey"`
	CreateTime  time.Time `gorm:"index"`
	Type        string
	MissionID   uint
	CreatorUID  string
	ContentUID  string `gorm:"index"`
	CotType     string
	Callsign    string
	IconsetPath string
	Color       string
	Lat         float64
	Lon         float64
}

func hasItem(items []string, item string) bool {
	for _, s := range items {
		if s == item {
			return true
		}
	}

	return false
}
