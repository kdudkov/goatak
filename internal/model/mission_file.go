package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/tools"
)

type Content struct {
	ID             uint `gorm:"primaryKey"`
	CreatedAt      time.Time
	Scope          string `gorm:"index"`
	Hash           string `gorm:"index"`
	UID            string `gorm:"uniqueIndex"`
	Name           string
	MIMEType       string
	Size           int
	SubmissionUser string
	CreatorUID     string
	Tool           string `gorm:"index"`
	Keywords       string
	Kw             tools.StringSet `gorm:"-"`
	MissionFiles   []*MissionFile  `gorm:"constraint:OnDelete:CASCADE"`
}

type MissionFile struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	MissionID  uint `gorm:"index"`
	ContentID  uint `gorm:"index"`
	CreatorUID string
	Content    *Content
}

func (c *Content) String() string {
	if c == nil {
		return "nil"
	}

	return fmt.Sprintf("file: %s, scope: %s, uid: %s, hash: %s", c.Name, c.Scope, c.UID, c.Hash)
}

func (c *MissionFile) String() string {
	if c == nil {
		return "nil"
	}

	return fmt.Sprintf("creator: %s, content: %s", c.CreatorUID, c.Content.String())
}

func (c *Content) BeforeSave(_ *gorm.DB) error {
	if c.UID == "" {
		c.UID = uuid.NewString()
	}

	if c.Kw != nil {
		c.Keywords = c.Kw.String()
	}

	return nil
}

func (c *Content) AfterFind(_ *gorm.DB) error {
	c.Kw = tools.StringToSet(c.Keywords)
	return nil
}
