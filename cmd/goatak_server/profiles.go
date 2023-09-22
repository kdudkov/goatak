package main

import (
	"os"
	"path/filepath"
	"strings"
)

func NewUserPrefsFile(callsign, team, role, typ string) *PrefFile {
	conf := NewUserProfilePrefFile()
	if callsign != "" {
		conf.AddParam("locationCallsign", callsign)
	}
	if team != "" {
		conf.AddParam("locationTeam", team)
	}
	if role != "" {
		conf.AddParam("atakRoleType", role)
	}
	if typ != "" {
		conf.AddParam("locationUnitType", typ)
	}
	return conf
}

func (app *App) GetProfile(user, uid string) []ZipFile {
	if app.users == nil {
		return nil
	}

	res := make([]ZipFile, 0)

	if user != "" {
		if userInfo := app.users.GetUser(user); userInfo != nil {
			if userInfo.Callsign != "" || userInfo.Team != "" || userInfo.Role != "" || userInfo.Typ != "" {
				res = append(res, NewUserPrefsFile(userInfo.Callsign, userInfo.Team, userInfo.Role, userInfo.Typ))
			}
		}
	}

	if f, err := NewFsFile("defaults.pref"); err == nil {
		res = append(res, f)
	}

	if paths, err := os.ReadDir(filepath.Join(baseDir, "maps")); err == nil {
		for _, p := range paths {
			if !p.IsDir() && strings.HasSuffix(p.Name(), ".xml") {
				if f, err := NewFsFile(filepath.Join("maps", p.Name())); err == nil {
					res = append(res, f)
				}
			}
		}
	}

	return res
}
