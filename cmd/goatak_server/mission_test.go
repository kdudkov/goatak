package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/kdudkov/goatak/cmd/goatak_server/database"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
)

func TestMissionSubscriptions(t *testing.T) {
	db := getTestDatabase()

	m := database.New(db)

	require.NoError(t, m.Migrate())

	m1 := &model.Mission{Name: "mission1", Scope: "s1"}
	m2 := &model.Mission{Name: "mission2", Scope: "s1"}

	require.NoError(t, m.CreateMission(m1))
	require.NoError(t, m.CreateMission(m2))

	require.NotEmpty(t, m1.ID)
	require.NotEmpty(t, m2.ID)

	user := &model.Device{Login: "login"}

	m.Subscribe(user, m1, "uid1", "")
	m.Subscribe(user, m1, "uid1", "")
	m.Subscribe(user, m1, "uid2", "")
	m.Subscribe(user, m2, "uid1", "")

	assert.Len(t, m.SubscriptionQuery().Mission(m1.ID).Get(), 2)
	assert.Len(t, m.GetSubscribers(m1.ID), 2)
	assert.Len(t, m.SubscriptionQuery().Mission(m2.ID).Get(), 1)
	assert.Len(t, m.GetSubscribers(m2.ID), 1)

	s1 := m.SubscriptionQuery().Mission(m1.ID).Client("uid1").One()
	assert.Equal(t, m1.ID, s1.MissionID)
	assert.Equal(t, "MISSION_SUBSCRIBER", s1.Role)
}

func TestMissionCRUD(t *testing.T) {
	db := getTestDatabase()

	m := database.New(db)
	require.NoError(t, m.Migrate())

	m1 := &model.Mission{Name: "mission1", Scope: "scope1"}
	m2 := &model.Mission{Name: "mission2", Scope: "scope1"}
	m3 := &model.Mission{Name: "mission1", Scope: "scope2"}

	require.NoError(t, m.CreateMission(m1))
	require.NoError(t, m.CreateMission(m2))
	require.NoError(t, m.CreateMission(m3))

	require.Error(t, m.CreateMission(&model.Mission{Name: "mission2", Scope: "scope1"}))

	assert.Len(t, m.MissionQuery().Scope("scope1").Full().Get(), 2)
	assert.Len(t, m.MissionQuery().Scope("scope1").Get(), 2)
	assert.Len(t, m.MissionQuery().Scope("scope2").Full().Get(), 1)
	assert.Len(t, m.MissionQuery().Scope("scope2").Get(), 1)
	assert.Empty(t, m.MissionQuery().Scope("no_scope").Full().Get())

	user := &model.Device{Login: "login"}

	m.Subscribe(user, m1, "uid1", "")
	m.Subscribe(user, m1, "uid1", "")
	m.Subscribe(user, m1, "uid2", "")
	m.Subscribe(user, m2, "uid1", "")

	assert.Len(t, m.SubscriptionQuery().Mission(m1.ID).Get(), 2)
	assert.Len(t, m.GetSubscribers(m1.ID), 2)
	assert.Len(t, m.SubscriptionQuery().Mission(m2.ID).Get(), 1)
	assert.Len(t, m.GetSubscribers(m2.ID), 1)

	m.MissionQuery().Delete(m2.ID)
	assert.Len(t, m.MissionQuery().Scope("scope1").Full().Get(), 1)

	assert.Len(t, m.SubscriptionQuery().Mission(m1.ID).Get(), 2)
	assert.Len(t, m.GetSubscribers(m1.ID), 2)
	assert.Empty(t, m.SubscriptionQuery().Mission(m2.ID).Get())
	assert.Empty(t, m.GetSubscribers(m2.ID))
}

func TestPointCRUD(t *testing.T) {
	db := getTestDatabase()

	m := database.New(db)
	require.NoError(t, m.Migrate())

	m1 := &model.Mission{Name: "mission1", Scope: "scope1"}
	m2 := &model.Mission{Name: "mission2", Scope: "scope1"}

	require.NoError(t, m.CreateMission(m1))
	require.NoError(t, m.CreateMission(m2))

	assert.NotNil(t, m.AddMissionPoint(m1, newCotMessage("scope1", "uid1", 10, 20)))
	assert.NotNil(t, m.AddMissionPoint(m1, newCotMessage("scope1", "uid2", 10, 20)))
	assert.Nil(t, m.AddMissionPoint(m1, newCotMessage("scope1", "uid1", 15, 20)))
	assert.NotNil(t, m.AddMissionPoint(m2, newCotMessage("scope1", "uid1", 15, 20)))

	require.Empty(t, m.MissionQuery().Scope("scope1").Name(m1.Name).One().Points)
	require.Len(t, m.MissionQuery().Scope("scope1").Name(m1.Name).Full().One().Points, 2)
	require.Len(t, m.MissionQuery().Scope("scope1").Name(m2.Name).Full().One().Points, 1)

	require.NotNil(t, m.DeleteMissionPoint(m1, "uid1", ""))
	require.Nil(t, m.DeleteMissionPoint(m1, "uid1", ""))

	require.Len(t, m.MissionQuery().Scope("scope1").Name(m1.Name).Full().One().Points, 1)
	require.Len(t, m.MissionQuery().Scope("scope1").Name(m2.Name).Full().One().Points, 1)

	ch := m.GetChanges(m1.ID, time.Now().Add(-time.Hour), false)
	fmt.Println(ch)
}

func TestMissionContent(t *testing.T) {
	db := getTestDatabase()

	m := database.New(db)
	require.NoError(t, m.Migrate())

	m1 := &model.Mission{Name: "mission1", Scope: "scope1"}
	require.NoError(t, m.CreateMission(m1))

	require.NoError(t, m.Save(&model.Resource{FileName: "file1", Hash: "aaa", Scope: "scope1"}))
	require.NoError(t, m.Save(&model.Resource{FileName: "file2", Hash: "bbb", Scope: "scope1"}))
	require.NoError(t, m.Save(&model.Resource{FileName: "file3", Hash: "ccc", Scope: "scope1"}))

	require.NotNil(t, m.AddMissionResource(m1, "aaa", "author"))
	require.Nil(t, m.AddMissionResource(m1, "aaa", "author"))
	require.NotNil(t, m.AddMissionResource(m1, "bbb", "author"))

	assert.Len(t, m1.Resources, 2)
	assert.NotNil(t, m1.Resources[0])

	m2 := m.MissionQuery().Id(m1.ID).Full().One()

	assert.Len(t, m2.Resources, 2)
	assert.NotNil(t, m2.Resources[0])
}

func getTestDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
	if err != nil {
		panic("failed to connect database")
	}

	return db
}

func getSubscription(missionId uint, uid string) *model.Subscription {
	return &model.Subscription{
		MissionID: missionId,
		ClientUID: uid,
		Username:  "aaa",
		Role:      "aaa",
	}
}

func newCotMessage(scope, uid string, lat, lon float64) *cot.CotMessage {
	tak := cot.BasicMsg("a-f-G", uid, time.Second*10)
	tak.CotEvent.Lat = lat
	tak.CotEvent.Lon = lon

	det, _ := cot.DetailsFromString(tak.GetCotEvent().GetDetail().GetXmlDetail())

	return &cot.CotMessage{TakMessage: tak, Detail: det, Scope: scope}
}
