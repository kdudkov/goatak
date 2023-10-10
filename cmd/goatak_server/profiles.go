package main

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func NewUserPrefsFile(prefix, callsign, team, role, typ string) *PrefFile {
	conf := NewUserProfilePrefFile(prefix)
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

func (app *App) GetProfileFiles(username, uid string) []FileContent {
	res := make([]FileContent, 0)
	prefix := fmt.Sprintf("%x", md5.Sum([]byte(username)))

	if app.users != nil && username != "" {
		if userInfo := app.users.GetUser(username); userInfo != nil {
			if userInfo.Callsign != "" || userInfo.Team != "" || userInfo.Role != "" || userInfo.Typ != "" {
				app.Logger.Debugf("add user prefs")
				f := NewUserPrefsFile(prefix, userInfo.Callsign, userInfo.Team, userInfo.Role, userInfo.Typ)
				res = append(res, f)
			}
		}
	}

	if f, err := NewFsFile(prefix+"/defaults.pref", filepath.Join(app.config.dataDir, "defaults.pref")); err == nil {
		app.Logger.Debugf("add default.prefs")
		res = append(res, f)
	}

	if paths, err := os.ReadDir(filepath.Join(app.config.dataDir, "maps")); err == nil {
		for _, p := range paths {
			if !p.IsDir() && strings.HasSuffix(p.Name(), ".xml") {
				if f, err := NewFsFile("maps/"+p.Name(), filepath.Join(app.config.dataDir, "maps", p.Name())); err == nil {
					app.Logger.Debugf("add %s", p.Name())
					res = append(res, f)
				}
			}
		}
	}
	return res
}
