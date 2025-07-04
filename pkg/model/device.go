package model

import (
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 14

type Device struct {
	Login       string   `gorm:"primaryKey;size:255" yaml:"user"`
	Password    string   `gorm:"not null;size:255" yaml:"password"`
	Scope       string   `gorm:"not null;size:255" yaml:"scope"`
	Disabled    bool     `gorm:"not null;default:false"`
	Admin       bool     `gorm:"not null;default:false"`
	ReadScope   []string `gorm:"serializer:json" yaml:"read_scope"`
	LastConnect *time.Time
	Certs       []*Certificate `gorm:"foreignKey:Login"`
}

type DeviceDTO struct {
	Login       string            `json:"login"`
	Scope       string            `json:"scope"`
	Disabled    bool              `json:"disabled"`
	Admin       bool              `json:"admin,omitempty"`
	ReadScope   []string          `json:"read_scope,omitempty"`
	LastConnect *time.Time        `json:"last_connect,omitempty"`
	Certs       []*CertificateDTO `json:"certs,omitempty"`
}

type DevicePutDTO struct {
	Admin     bool     `json:"admin,omitempty"`
	Disabled  bool     `json:"disabled"`
	Password  string   `json:"password,omitempty"`
	Scope     string   `json:"scope,omitempty"`
	ReadScope []string `json:"read_scope,omitempty"`
}

type DevicePostDTO struct {
	Login string `json:"login,omitempty"`
	DevicePutDTO
}

func (u *Device) GetLogin() string {
	if u == nil {
		return ""
	}

	return u.Login
}

func (u *Device) GetScope() string {
	if u == nil {
		return ""
	}

	return u.Scope
}

func (u *Device) GetReadScope() []string {
	if u == nil {
		return nil
	}

	return u.ReadScope
}

func (u *Device) CanSeeScope(scope string) bool {
	// nil user can see empty scope (no auth mode)
	if u == nil {
		return scope == ""
	}

	if u.GetScope() == scope {
		return true
	}

	for _, s := range u.ReadScope {
		if s == "*" || s == scope {
			return true
		}
	}

	return false
}

func (u *Device) CheckPassword(password string) bool {
	if u == nil {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		slog.Debug("password check failed", slog.Any("error", err))
		return false
	}

	return true
}

func (u *Device) SetPassword(password string) error {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return err
	}

	u.Password = string(b)

	return nil
}

func (u *Device) CanLogIn() bool {
	return u.IsGood() && u.Admin
}

func (u *Device) IsGood() bool {
	if u == nil {
		return false
	}

	return !u.Disabled
}

func (u *Device) DTO() *DeviceDTO {
	if u == nil {
		return nil
	}

	certs := make([]*CertificateDTO, len(u.Certs))
	for i, c := range u.Certs {
		certs[i] = c.DTO()
	}

	return &DeviceDTO{
		Login:       u.Login,
		Scope:       u.Scope,
		Disabled:    u.Disabled,
		Admin:       u.Admin,
		ReadScope:   u.ReadScope,
		LastConnect: u.LastConnect,
		Certs:       certs,
	}
}
