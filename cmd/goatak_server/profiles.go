package main

import (
	"os"
	"path/filepath"
	"strings"

	"maps"

	"github.com/kdudkov/goatak/cmd/goatak_server/mp"
	"github.com/kdudkov/goatak/pkg/model"
)

var defaultPrefs map[string]string = map[string]string{
	"deviceProfileEnableOnConnect":  "true",
	"speed_unit_pref":               "1",
	"alt_unit_pref":                 "1",
	"saHasPhoneNumber":              "false",
	"alt_display_pref":              "MSL",
	"coord_display_pref":            "DM",
	"rab_north_ref_pref":            "1",
	"rab_brg_units":                 "0",
	"rab_nrg_units":                 "1",
	"displayServerConnectionWidget": "true",
	"frame_limit": "1",
	"hidePreferenceItem_deviceProfileEnableOnConnect": "true",
}

func profileOpts(profiles ...*model.Profile) map[string]string {
	res := make(map[string]string)

	maps.Copy(res, defaultPrefs)

	for _, p := range profiles {
		if p == nil {
			continue
		}

		if p.Callsign != "" {
			res["locationCallsign"] = p.Callsign
		}

		if p.Team != "" {
			res["locationTeam"] = p.Team
		}

		if p.Role != "" {
			res["atakRoleType"] = p.Role
		}

		if p.CotType != "" {
			res["locationUnitType"] = p.CotType
		}

		maps.Copy(res, p.Options)
	}

	return res
}

func (app *App) GetProfileFiles(username, uid string) []mp.FileContent {
	res := make([]mp.FileContent, 0)

	options := profileOpts(
		app.dbm.ProfileQuery().Login(username).UID("").One(),
		app.dbm.ProfileQuery().Login(username).UID(uid).One(),
	)

	if len(options) > 0 {
		conf := mp.NewPrefFile("user-profile.pref")
		for k, v := range options {
			conf.AddParam(mp.CIV_PREF, k, v)
		}

		res = append(res, conf)
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
