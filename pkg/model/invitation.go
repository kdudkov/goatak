package model

import "time"

type Invitation struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	MissionID  uint `gorm:"index;not null"`
	Mission    *Mission
	Typ        string `gorm:"index;not null"`
	Invitee    string `gorm:"index;not null"`
	CreatorUID string
	Role       string
}
