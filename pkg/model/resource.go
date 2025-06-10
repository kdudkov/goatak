package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/util"
)

type Resource struct {
	ID             uint `gorm:"primaryKey"`
	CreatedAt      time.Time
	Scope          string `gorm:"index;not null;size:255"`
	Hash           string `gorm:"index;size:255"`
	UID            string `gorm:"uniqueIndex;size:255"`
	Name           string `gorm:"size:255"`
	FileName       string `gorm:"size:255"`
	MIMEType       string `gorm:"size:255"`
	Size           int
	SubmissionUser string `gorm:"size:255"`
	CreatorUID     string `gorm:"size:255"`
	Tool           string `gorm:"index"`
	Keywords       string
	Groups         string
	Expiration     int64
	KwSet          util.StringSet `gorm:"-"`
}

func (c *Resource) String() string {
	if c == nil {
		return "nil"
	}

	return fmt.Sprintf("file: %s, scope: %s, uid: %s, hash: %s", c.FileName, c.Scope, c.UID, c.Hash)
}

func (c *Resource) BeforeSave(_ *gorm.DB) error {
	if c.UID == "" {
		c.UID = uuid.NewString()
	}

	if c.KwSet != nil {
		c.Keywords = c.KwSet.String()
	}

	return nil
}

func (c *Resource) AfterFind(_ *gorm.DB) error {
	c.KwSet = util.StringToSet(c.Keywords)
	return nil
}
