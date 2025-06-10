package model

import (
	"fmt"
	"time"
)

const (
	CHANGE_TYPE_ADD    = "ADD_CONTENT"
	CHANGE_TYPE_REMOVE = "REMOVE_CONTENT"
)

type Change struct {
	ID             uint      `gorm:"primaryKey"`
	CreatedAt      time.Time `gorm:"index"`
	Type           string    `gorm:"size:255"`
	MissionID      uint      `gorm:"index;not null"`
	CreatorUID     string    `gorm:"size:255"`
	ContentUID     string    `gorm:"size:255"`
	MissionPointID *uint
	MissionPoint   *Point `gorm:"foreignKey:MissionPointID"`
	ContentHash    string `gorm:"size:255"`
	ResourceID     *uint
	Resource       *Resource `gorm:"foreignKey:ResourceID"`
}

func (c *Change) String() string {
	if c == nil {
		return "nil"
	}

	if c.MissionPointID != nil {
		return fmt.Sprintf("POINT %s, mid: %d, uid: %s, %d", c.Type, c.MissionID, c.ContentUID, c.MissionPointID)
	}

	if c.ResourceID != nil {
		return fmt.Sprintf("RESOURCE %s, mid: %d, uid: %s, %d", c.Type, c.MissionID, c.ContentUID, c.ResourceID)
	}

	return fmt.Sprintf("INVALID %s, mid: %d, uid: %s", c.Type, c.MissionID, c.ContentUID)
}
