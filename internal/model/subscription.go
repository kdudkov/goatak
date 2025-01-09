package model

import "time"

type Subscription struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	MissionID uint   `gorm:"index"`
	ClientUID string `gorm:"index"`
	Username  string
	Role      string
}
