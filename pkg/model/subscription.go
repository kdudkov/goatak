package model

import "time"

type Subscription struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	MissionID uint   `gorm:"index;not null"`
	ClientUID string `gorm:"index;not null"`
	Username  string
	Role      string
}
