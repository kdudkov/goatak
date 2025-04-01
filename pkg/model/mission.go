package model

import (
	"time"
)

type Mission struct {
	ID             uint `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Scope          string `gorm:"index;not null"`
	Name           string `gorm:"index;not null"`
	Creator        string
	CreatorUID     string
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
	Resources      []*Resource `gorm:"many2many:mission_resources;"`
	Points         []*Point    `gorm:"many2many:mission_points;"`
	Token          string
}
