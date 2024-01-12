package model

import (
	"time"
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
	Items          []DataItem
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
	Event       []byte
}
