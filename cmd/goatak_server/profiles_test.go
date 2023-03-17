package main

import (
	"github.com/google/uuid"
	"os"
	"testing"
)

func TestMissionPackage_Create(t *testing.T) {
	mp := NewMissionPackage("ProfileMissionPackage-"+uuid.New().String(), "Enrollment")

	mp.Param("onReceiveImport", "true")
	mp.Param("onReceiveDelete", "true")

	conf := NewUserProfilePrefFile()
	conf.AddParam("locationCallsign", "TestUser")
	conf.AddParam("locationTeam", "Cyan")
	conf.AddParam("atakRoleType", "Medic")

	mp.AddFile(conf)

	f, _ := os.Create("/tmp/profile.zip")

	dat, err := mp.Create()

	if err != nil {
		t.Error(err)
	}
	f.Write(dat)
	f.Close()
}
