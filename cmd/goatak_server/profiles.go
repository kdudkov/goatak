package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kdudkov/goatak/cmd/goatak_server/mp"
	"github.com/kdudkov/goatak/pkg/model"
)

func NewUserPrefsFile(user *model.Device) *mp.PrefFile {
	conf := mp.NewPrefFile("user-profile.pref")
	if user.Callsign != "" {
		conf.AddParam(mp.CIV_PREF, "locationCallsign", user.Callsign)
	}

	if user.Team != "" {
		conf.AddParam(mp.CIV_PREF, "locationTeam", user.Team)
	}

	if user.Role != "" {
		conf.AddParam(mp.CIV_PREF, "atakRoleType", user.Role)
	}

	if user.CotType != "" {
		conf.AddParam(mp.CIV_PREF, "locationUnitType", user.CotType)
	}

	conf.AddParam(mp.CIV_PREF, "deviceProfileEnableOnConnect", true)
	conf.AddParam(mp.CIV_PREF, "coord_display_pref", "DD")

	for k, v := range user.Options {
		conf.AddParam(mp.CIV_PREF, k, v)
	}

	return conf
}

func (app *App) GetProfileFiles(username, uid string) []mp.FileContent {
	res := make([]mp.FileContent, 0)

	if userInfo := app.users.Get(username); userInfo != nil {
		if userInfo.HasProfile() {
			app.logger.Debug("add user prefs")
			res = append(res, NewUserPrefsFile(userInfo))
		}
	}

	if f, err := mp.NewFsFile("defaults.pref", filepath.Join(app.config.DataDir(), "defaults.pref")); err == nil {
		app.logger.Debug("add default.prefs")
		res = append(res, f)
	}

	if paths, err := os.ReadDir(filepath.Join(app.config.DataDir(), "maps")); err == nil {
		for _, p := range paths {
			if !p.IsDir() && strings.HasSuffix(p.Name(), ".xml") {
				if f, err := mp.NewFsFile("maps/"+p.Name(), filepath.Join(app.config.DataDir(), "maps", p.Name())); err == nil {
					app.logger.Debug("add " + p.Name())

					res = append(res, f)
				}
			}
		}
	}

	return res
}
