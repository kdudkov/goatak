package model

import (
	"database/sql"
	"fmt"
	"time"
)

type Change struct {
	ID             uint      `gorm:"primaryKey"`
	CreatedAt      time.Time `gorm:"index"`
	Type           string
	MissionID      uint `gorm:"index"`
	CreatorUID     string
	ContentUID     string
	MissionPointID sql.NullInt32
	MissionPoint   *Point `gorm:"foreignKey:MissionPointID"`
	ContentHash    string
	ResourceID     sql.NullInt32
	Resource       *Resource `gorm:"foreignKey:ResourceID"`
}

func (c *Change) String() string {
	if c == nil {
		return "nil"
	}

	if c.MissionPointID.Valid {
		return fmt.Sprintf("POINT %s, mid: %d, uid: %s, %d", c.Type, c.MissionID, c.ContentUID, c.MissionPointID.Int32)
	}

	if c.ResourceID.Valid {
		return fmt.Sprintf("RESOURCE %s, mid: %d, uid: %s, %d", c.Type, c.MissionID, c.ContentUID, c.ResourceID.Int32)
	}

	return fmt.Sprintf("INVALID %s, mid: %d, uid: %s", c.Type, c.MissionID, c.ContentUID)
}
