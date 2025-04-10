package mp

import (
	"os"
	"testing"

	"github.com/google/uuid"
)

func TestMissionPackage_Create(t *testing.T) {
	mp := NewMissionPackage("ProfileMissionPackage-"+uuid.New().String(), "Enrollment")

	mp.Param("onReceiveImport", "true")
	mp.Param("onReceiveDelete", "true")

	conf := NewUserProfilePrefFile("aaa")
	conf.AddParam(CIV_PREF, "locationCallsign", "TestUser")
	conf.AddParam(CIV_PREF, "locationTeam", "Cyan")
	conf.AddParam(CIV_PREF, "atakRoleType", "Medic")

	mp.AddFile(conf)

	dat, err := mp.Create()
	if err != nil {
		t.Error(err)
	}

	if len(dat) == 0 {
		t.Error("no data")
	}

	f, _ := os.Create("/tmp/profile.zip")
	_, _ = f.Write(dat)
	_ = f.Close()
}
