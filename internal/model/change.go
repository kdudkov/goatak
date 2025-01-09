package model

import (
	"fmt"
	"time"
)

type Change struct {
	ID          uint      `gorm:"primaryKey"`
	CreatedAt   time.Time `gorm:"index"`
	Type        string
	MissionID   uint `gorm:"index"`
	CreatorUID  string
	ContentUID  string `gorm:"index"`
	CotType     string
	Callsign    string
	IconsetPath string
	Color       string
	Lat         float64
	Lon         float64
}

func (c *Change) String() string {
	if c == nil {
		return "nil"
	}

	return fmt.Sprintf("%s, mid: %d, uid: %s", c.Type, c.MissionID, c.ContentUID)
}
