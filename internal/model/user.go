package model

type User struct {
	Login    string `yaml:"user"`
	Callsign string `yaml:"callsign,omitempty"`
	Team     string `yaml:"team,omitempty"`
	Role     string `yaml:"role,omitempty"`
	Typ      string `yaml:"type,omitempty"`
	Password string `yaml:"password"`
	Scope    string `yaml:"scope,omitempty"`
}
