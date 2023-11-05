package model

type User struct {
	Login     string   `yaml:"user"`
	Callsign  string   `yaml:"callsign,omitempty"`
	Team      string   `yaml:"team,omitempty"`
	Role      string   `yaml:"role,omitempty"`
	Typ       string   `yaml:"type,omitempty"`
	Password  string   `yaml:"password"`
	Scope     string   `yaml:"scope,omitempty"`
	ReadScope []string `yaml:"read_scope"`
}

func (u *User) GetLogin() string {
	if u == nil {
		return ""
	}
	return u.Login
}

func (u *User) GetScope() string {
	if u == nil {
		return ""
	}
	return u.Scope
}

func (u *User) CanSeeScope(scope string) bool {
	if u == nil {
		return true
	}
	if u.Scope == "" || u.Scope == scope {
		return true
	}

	for _, s := range u.ReadScope {
		if s == "*" || s == scope {
			return true
		}
	}

	return false
}
