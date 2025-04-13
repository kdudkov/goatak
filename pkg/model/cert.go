package model

import (
	"time"
)

type Certificate struct {
	UID         string `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Login       string `gorm:"not null;index"`
	Serial      string `gorm:"not null;index"`
	LastConnect *time.Time
	ValidTill   *time.Time
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
