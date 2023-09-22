package model

type UserInfo struct {
	User     string `yaml:"user"`
	Callsign string `yaml:"callsign"`
	Team     string `yaml:"team"`
	Role     string `yaml:"role"`
	Typ      string `yaml:"type"`
	Password string `yaml:"password"`
	Scope    string `yaml:"scope"`
}
