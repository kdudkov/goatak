package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/tools"
)

type Resource struct {
	ID             uint `gorm:"primaryKey"`
	CreatedAt      time.Time
	Scope          string `gorm:"index"`
	Hash           string `gorm:"index"`
	UID            string `gorm:"uniqueIndex"`
	FileName       string
	MIMEType       string
	Size           int
	SubmissionUser string
	CreatorUID     string
	Tool           string `gorm:"index"`
	Keywords       string
	Groups         string
	Expiration     int64
	Kw             tools.StringSet `gorm:"-"`
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

	if c.Kw != nil {
		c.Keywords = c.Kw.String()
	}

	return nil
}

func (c *Resource) AfterFind(_ *gorm.DB) error {
	c.Kw = tools.StringToSet(c.Keywords)
	return nil
}
