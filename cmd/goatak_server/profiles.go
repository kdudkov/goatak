package main

import (
	"os"
	"path/filepath"
	"strings"

	"maps"

	"github.com/kdudkov/goatak/cmd/goatak_server/mp"
	"github.com/kdudkov/goatak/pkg/model"
)

func profileOpts(callsign bool, profiles ...*model.Profile) map[string]string {
	res := make(map[string]string)

	for _, p := range profiles {
		if p == nil {
			continue
		}

		if callsign {
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
		}

		maps.Copy(res, p.Options)
	}

	return res
}

func (app *App) GetProfileFiles(username, uid string, enrollment bool) []mp.FileContent {
	res := make([]mp.FileContent, 0)

	options := profileOpts(enrollment,
		app.dbm.ProfileQuery().Login("*").UID("*").One(),
		app.dbm.ProfileQuery().Login(username).UID("*").One(),
	)

	maps.Copy(options,
		profileOpts(true,
			app.dbm.ProfileQuery().Login("*").UID(uid).One(),
			app.dbm.ProfileQuery().Login(username).UID(uid).One(),
		),
	)

	options["deviceProfileEnableOnConnect"] = "true"

	if len(options) > 0 {
		conf := mp.NewPrefFile("user-profile.pref")
		for k, v := range options {
			conf.AddParam(mp.CIV_PREF, k, v)
		}

		res = append(res, conf)
	}

	if !enrollment {
		// add maps only for enrollment
		return res
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
