package model

type User struct {
	Login     string   `yaml:"user"`
	Callsign  string   `yaml:"callsign,omitempty"`
	Team      string   `yaml:"team,omitempty"`
	Role      string   `yaml:"role,omitempty"`
	Typ       string   `yaml:"type,omitempty"`
	Password  string   `yaml:"password"`
	Scope     string   `yaml:"scope"`
	ReadScope []string `yaml:"read_scope,omitempty"`
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

func (u *User) HasProfile() bool {
	if u == nil {
		return false
	}

	return u.Callsign != "" || u.Team != "" || u.Role != "" || u.Typ != ""
}
