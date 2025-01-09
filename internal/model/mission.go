package model

import (
	"time"
)

type Mission struct {
	ID             uint `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Scope          string `gorm:"index"`
	Name           string `gorm:"index"`
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
	Points         []*MissionPoint `gorm:"constraint:OnDelete:CASCADE"`
	Files          []*MissionFile  `gorm:"constraint:OnDelete:CASCADE"`
	Token          string
}
