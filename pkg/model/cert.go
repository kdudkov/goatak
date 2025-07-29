package model

import (
	"time"
)

type Certificate struct {
	Serial      string     `gorm:"primaryKey;size:255"`
	CreatedAt   time.Time  `gorm:"index;type:timestamp"`
	UpdatedAt   time.Time  `gorm:"type:timestamp"`
	Login       string     `gorm:"not null;index;size:255"`
	UID         string     `gorm:"index;size:255"`
	LastConnect *time.Time `gorm:"type:timestamp"`
	ValidTill   *time.Time `gorm:"type:timestamp"`
}

type CertificateDTO struct {
	UID         string     `json:"uid"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Login       string     `json:"login"`
	Serial      string     `json:"serial"`
	LastConnect *time.Time `json:"last_connect"`
}

func (c *Certificate) DTO() *CertificateDTO {
	if c == nil {
		return nil
	}

	return &CertificateDTO{
		UID:         c.UID,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Login:       c.Login,
		Serial:      c.Serial,
		LastConnect: c.LastConnect,
	}
}
