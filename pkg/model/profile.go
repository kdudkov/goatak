package model

type Profile struct {
	Login    string            `gorm:"primaryKey;size:255"`
	UID      string            `gorm:"primaryKey;size:255"`
	Callsign string            `gorm:"size:255"`
	Team     string            `gorm:"size:255"`
	Role     string            `gorm:"size:255"`
	CotType  string            `gorm:"size:255"`
	Options  map[string]string `gorm:"serializer:json"`
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
