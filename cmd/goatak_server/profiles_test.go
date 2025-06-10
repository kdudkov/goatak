package main

import (
	"testing"

	"github.com/kdudkov/goatak/internal/database"
	"github.com/kdudkov/goatak/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestProfiles(t *testing.T) {
	db := getTestDatabase()
	m := database.New(db)
	require.NoError(t, m.Migrate())
	
	username := "test"
	uid := "test_uid"
	
	m.Create(&model.Profile{
		Login: username,  
		Callsign: "aaa", 
		Team: "Red", 
		Options: map[string]string{"dexControls": "true"},
	})
	
	m.Create(&model.Profile{
		Login: username, 
		UID: uid, 
		Callsign: "bbb", 
		Options: map[string]string{"dexControls": "false"},
	})
	
	options := profileOpts(
		m.ProfileQuery().Login("").UID("").One(),
		m.ProfileQuery().Login(username).UID("").One(),
		m.ProfileQuery().Login(username).UID(uid).One(),
	)

	require.Len(t, options, len(defaultPrefs) + 3)
	require.Equal(t, "Red", options["locationTeam"])
	require.Equal(t, "bbb", options["locationCallsign"])
	require.Equal(t, "false", options["dexControls"])
}