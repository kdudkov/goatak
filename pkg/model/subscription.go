package model

import "time"

type Subscription struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	MissionID uint   `gorm:"index;not null"`
	ClientUID string `gorm:"index;not null;size:255"`
	Username  string `gorm:"size:255"`
	Role      string `gorm:"size:255"`
}
