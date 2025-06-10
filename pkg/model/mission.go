package model

import (
	"time"
)

type Mission struct {
	ID             uint `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Scope          string `gorm:"index;not null;size:255"`
	Name           string `gorm:"index;not null;size:255"`
	Creator        string `gorm:"size:255"`
	CreatorUID     string `gorm:"size:255"`
	BaseLayer      string
	Bbox           string
	ChatRoom       string
	Classification string
	Description    string `gorm:"size:255"`
	InviteOnly     bool
	Password       string `gorm:"size:255"`
	Path           string
	Tool           string `gorm:"size:255"`
	Groups         string
	Keywords       string
	Resources      []*Resource `gorm:"many2many:mission_resources;"`
	Points         []*Point    `gorm:"many2many:mission_points;"`
	Token          string
}
