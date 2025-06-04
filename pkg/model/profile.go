package model

type Profile struct {
	Login    string            `gorm:"primaryKey"`
	UID      string            `gorm:"primaryKey"`
	Callsign string            `gorm:"not null;default:''"`
	Team     string            `gorm:"not null;default:''"`
	Role     string            `gorm:"not null;default:''"`
	CotType  string            `gorm:"not null;default:''"`
	Options  map[string]string `gorm:"serializer:json" yaml:"options,omitempty"`
}

type ProfileDTO struct {
	Login    string            `json:"login"`
	UID      string            `json:"uid"`
	Callsign string            `json:"callsign,omitempty"`
	Team     string            `json:"team,omitempty"`
	Role     string            `json:"role,omitempty"`
	CotType  string            `json:"cot_type,omitempty"`
	Options  map[string]string `json:"options,omitempty"`
}

type ProfilePutDTO struct {
	Callsign string            `json:"callsign,omitempty"`
	Team     string            `json:"team,omitempty"`
	Role     string            `json:"role,omitempty"`
	CotType  string            `json:"cot_type,omitempty"`
	Options  map[string]string `json:"options,omitempty"`
}

type ProfilePostDTO struct {
	Login string `json:"login,omitempty"`
	UID   string `json:"uid,omitempty"`
	ProfilePutDTO
}

func (p *Profile) DTO() *ProfileDTO {
	if p == nil {
		return nil
	}

	return &ProfileDTO{
		Login:    p.Login,
		UID:      p.UID,
		Callsign: p.Callsign,
		Team:     p.Team,
		Role:     p.Role,
		CotType:  p.CotType,
		Options:  p.Options,
	}
}
