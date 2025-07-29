package model

import (
	"time"
)

type Mission struct {
	ID             uint      `gorm:"primaryKey"`
	CreatedAt      time.Time `gorm:"type:timestamp"`
	UpdatedAt      time.Time `gorm:"type:timestamp"`
	Scope          string    `gorm:"index;not null;size:255"`
	Name           string    `gorm:"index;not null;size:255"`
	Creator        string    `gorm:"size:255"`
	CreatorUID     string    `gorm:"size:255"`
	BaseLayer      string    `gorm:"size:255"`
	Bbox           string    `gorm:"size:255"`
	ChatRoom       string    `gorm:"size:255"`
	Classification string    `gorm:"size:255"`
	Description    string    `gorm:"size:255"`
	InviteOnly     bool
	Password       string      `gorm:"size:255"`
	Path           string      `gorm:"size:255"`
	Tool           string      `gorm:"size:255"`
	Groups         string      `gorm:"size:255"`
	Keywords       string      `gorm:"size:255"`
	Resources      []*Resource `gorm:"many2many:mission_resources;"`
	Points         []*Point    `gorm:"many2many:mission_points;"`
	Token          string      `gorm:"size:255"`
}
