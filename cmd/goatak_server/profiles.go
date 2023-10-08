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

func (app *App) GetProfileFiles(user, uid string) []FileContent {
	res := make([]FileContent, 0)

	if app.users != nil && user != "" {
		if userInfo := app.users.GetUser(user); userInfo != nil {
			if userInfo.Callsign != "" || userInfo.Team != "" || userInfo.Role != "" || userInfo.Typ != "" {
				app.Logger.Debugf("add user prefs")
				res = append(res, NewUserPrefsFile(userInfo.Callsign, userInfo.Team, userInfo.Role, userInfo.Typ))
			}
		}
	}

	if f, err := NewFsFile("defaults.pref", filepath.Join(app.config.dataDir, "defaults.pref")); err == nil {
		app.Logger.Debugf("add default.prefs")
		res = append(res, f)
	}

	if paths, err := os.ReadDir(filepath.Join(app.config.dataDir, "maps")); err == nil {
		for _, p := range paths {
			if !p.IsDir() && strings.HasSuffix(p.Name(), ".xml") {
				if f, err := NewFsFile(p.Name(), filepath.Join(app.config.dataDir, "maps", p.Name())); err == nil {
					app.Logger.Debugf("add %s", p.Name())
					res = append(res, f)
				}
			}
		}
	}
	return res
}
