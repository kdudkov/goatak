package model

import "time"

type Invitation struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	MissionID  uint `gorm:"index"`
	Mission    *Mission
	Typ        string `gorm:"index"`
	Invitee    string `gorm:"index"`
	CreatorUID string
	Role       string
}
