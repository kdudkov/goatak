package model

import (
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 14

type Device struct {
	Login       string         `gorm:"primaryKey" yaml:"user"`
	Callsign    string         `gorm:"not null;default:''" yaml:"callsign,omitempty"`
	Team        string         `gorm:"not null;default:''" yaml:"team,omitempty"`
	Role        string         `gorm:"not null;default:''" yaml:"role,omitempty"`
	CotType     string         `gorm:"not null;default:''" yaml:"type,omitempty"`
	Password    string         `gorm:"not null" yaml:"password"`
	Scope       string         `gorm:"not null" yaml:"scope"`
	Disabled    bool           `gorm:"not null;default:false"`
	ReadScope   []string       `gorm:"serializer:json" yaml:"read_scope,omitempty"`
	Options     map[string]any `gorm:"serializer:json" yaml:"options,omitempty"`
	LastSign    *time.Time
	LastConnect *time.Time
	Serial      string `gorm:"not null;default:''"`
	UID         string `gorm:"not null;default:''"`
}

type DeviceDTO struct {
	Login       string         `json:"login"`
	Callsign    string         `json:"callsign,omitempty"`
	Team        string         `json:"team,omitempty"`
	Role        string         `json:"role,omitempty"`
	CotType     string         `json:"cot_type,omitempty"`
	Scope       string         `json:"scope,omitempty"`
	Disabled    bool           `json:"disabled"`
	ReadScope   []string       `json:"read_scope,omitempty"`
	Options     map[string]any `json:"options,omitempty"`
	LastSign    *time.Time     `json:"last_sign,omitempty"`
	LastConnect *time.Time     `json:"last_connect,omitempty"`
	Serial      string         `json:"serial,omitempty"`
	UID         string         `json:"uid,omitempty"`
}

type DevicePutDTO struct {
	Callsign  string   `json:"callsign,omitempty"`
	Password  string   `json:"password,omitempty"`
	Team      string   `json:"team,omitempty"`
	Role      string   `json:"role,omitempty"`
	CotType   string   `json:"cot_type,omitempty"`
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

func (u *Device) HasProfile() bool {
	if u == nil {
		return false
	}

	return u.Callsign != "" || u.Team != "" || u.Role != "" || u.CotType != "" || len(u.Options) > 0
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

func (u *Device) DTO() *DeviceDTO {
	if u == nil {
		return nil
	}

	return &DeviceDTO{
		Login:       u.Login,
		Callsign:    u.Callsign,
		Team:        u.Team,
		Role:        u.Role,
		CotType:     u.CotType,
		Scope:       u.Scope,
		Disabled:    u.Disabled,
		ReadScope:   u.ReadScope,
		Options:     u.Options,
		LastSign:    u.LastSign,
		LastConnect: u.LastConnect,
		Serial:      u.Serial,
		UID:         u.UID,
	}
}
