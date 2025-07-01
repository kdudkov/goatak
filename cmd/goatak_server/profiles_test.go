package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kdudkov/goatak/cmd/goatak_server/mp"
	"github.com/kdudkov/goatak/internal/database"
	"github.com/kdudkov/goatak/pkg/model"
)

func TestProfiles(t *testing.T) {
	db := getTestDatabase()
	m := database.New(db)
	require.NoError(t, m.Migrate())

	username := "test"
	uid := "test_uid"

	m.Create(&model.Profile{
		Login: "*",
		UID:   "*",
		Options: map[string]string{
			"team_color_gps_icon": "true",
			"coord_display_pref":  "DD",
		},
	})

	m.Create(&model.Profile{
		Login:    username,
		UID:      "*",
		Callsign: "aaa",
		Team:     "Red",
		Options:  map[string]string{"dexControls": "true"},
	})

	m.Create(&model.Profile{
		Login:    username,
		UID:      uid,
		Callsign: "bbb",
		Options:  map[string]string{"dexControls": "false"},
	})

	options := profileOpts(
		m.ProfileQuery().Login("*").UID("*").One(),
		m.ProfileQuery().Login(username).UID("*").One(),
		m.ProfileQuery().Login(username).UID(uid).One(),
	)

	require.Len(t, options, 5)
	require.Equal(t, "Red", options["locationTeam"])
	require.Equal(t, "bbb", options["locationCallsign"])
	require.Equal(t, "false", options["dexControls"])
	require.Equal(t, "true", options["team_color_gps_icon"])
	require.Equal(t, "DD", options["coord_display_pref"])

	conf := mp.NewPrefFile("user-profile.pref")
	for k, v := range options {
		conf.AddParam(mp.CIV_PREF, k, v)
	}

	fmt.Println(string(conf.Content()))
}
