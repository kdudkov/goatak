package model

import "time"

type Invitation struct {
	ID         uint      `gorm:"primaryKey"`
	CreatedAt  time.Time `gorm:"type:timestamp"`
	MissionID  uint      `gorm:"index;not null"`
	Mission    *Mission
	Typ        string `gorm:"index;not null;size:255"`
	Invitee    string `gorm:"index;not null;size:255"`
	CreatorUID string `gorm:"size:255"`
	Role       string `gorm:"size:255"`
}
