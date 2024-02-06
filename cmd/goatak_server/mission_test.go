package main

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/cot"
)

func TestMissionSubscriptions(t *testing.T) {
	db := prepare()

	m := NewMissionManager(db, nil)
	require.NoError(t, m.Migrate())

	m1 := &model.Mission{Name: "mission1", Scope: "s1"}
	m2 := &model.Mission{Name: "mission2", Scope: "s1"}

	require.NoError(t, m.PutMission(m1))
	require.NoError(t, m.PutMission(m2))

	require.NotEmpty(t, m1.ID)
	require.NotEmpty(t, m2.ID)

	m.PutSubscription(getSubscription(m1.ID, "uid1"))
	m.PutSubscription(getSubscription(m1.ID, "uid1"))
	m.PutSubscription(getSubscription(m1.ID, "uid2"))
	m.PutSubscription(getSubscription(m2.ID, "uid1"))

	assert.Len(t, m.GetSubscriptions(m1.ID), 2)
	assert.Len(t, m.GetSubscribers(m1.ID), 2)
	assert.Len(t, m.GetSubscriptions(m2.ID), 1)
	assert.Len(t, m.GetSubscribers(m2.ID), 1)

	s1 := m.GetSubscription(m1.ID, "uid1")
	assert.Equal(t, m1.ID, s1.MissionID)
	assert.Equal(t, "aaa", s1.Role)
}

func TestMissionCRUD(t *testing.T) {
	db := prepare()

	m := NewMissionManager(db, nil)
	require.NoError(t, m.Migrate())

	m1 := &model.Mission{Name: "mission1", Scope: "scope1"}
	m2 := &model.Mission{Name: "mission2", Scope: "scope1"}
	m3 := &model.Mission{Name: "mission1", Scope: "scope2"}

	require.NoError(t, m.PutMission(m1))
	require.NoError(t, m.PutMission(m2))
	require.NoError(t, m.PutMission(m3))

	require.Error(t, m.PutMission(&model.Mission{Name: "mission2", Scope: "scope1"}))

	assert.Len(t, m.GetAllMissions("scope1"), 2)
	assert.Len(t, m.GetAllMissions("scope2"), 1)
	assert.Len(t, m.GetAllMissions("scope3"), 0)

	m.PutSubscription(getSubscription(m1.ID, "uid1"))
	m.PutSubscription(getSubscription(m1.ID, "uid1"))
	m.PutSubscription(getSubscription(m1.ID, "uid2"))
	m.PutSubscription(getSubscription(m2.ID, "uid1"))

	assert.Len(t, m.GetSubscriptions(m1.ID), 2)
	assert.Len(t, m.GetSubscribers(m1.ID), 2)
	assert.Len(t, m.GetSubscriptions(m2.ID), 1)
	assert.Len(t, m.GetSubscribers(m2.ID), 1)

	m.DeleteMission(m2.ID)
	assert.Len(t, m.GetAllMissions("scope1"), 1)

	assert.Len(t, m.GetSubscriptions(m1.ID), 2)
	assert.Len(t, m.GetSubscribers(m1.ID), 2)
	assert.Empty(t, m.GetSubscriptions(m2.ID))
	assert.Empty(t, m.GetSubscribers(m2.ID))
}

func TestAddPoint(t *testing.T) {
	db := prepare()

	m := NewMissionManager(db, nil)
	require.NoError(t, m.Migrate())

	m1 := &model.Mission{Name: "mission1", Scope: "scope1"}
	m2 := &model.Mission{Name: "mission2", Scope: "scope1"}

	require.NoError(t, m.PutMission(m1))
	require.NoError(t, m.PutMission(m2))

	assert.True(t, m.AddPoint(m1, newCotMessage("scope1", "uid1", 10, 20)))
	assert.True(t, m.AddPoint(m1, newCotMessage("scope1", "uid2", 10, 20)))
	assert.False(t, m.AddPoint(m1, newCotMessage("scope1", "uid1", 15, 20)))
	assert.True(t, m.AddPoint(m2, newCotMessage("scope1", "uid1", 15, 20)))

	assert.Len(t, m.GetMission("scope1", m1.Name).Items, 2)
	assert.Len(t, m.GetMission("scope1", m2.Name).Items, 1)

	assert.True(t, m.DeleteMissionPoint(m1.ID, "uid1", ""))
	assert.False(t, m.DeleteMissionPoint(m1.ID, "uid1", ""))

	assert.Len(t, m.GetMission("scope1", m1.Name).Items, 1)
	assert.Len(t, m.GetMission("scope1", m2.Name).Items, 1)
}

func TestHash(t *testing.T) {
	m := &model.Mission{Name: "mission1", Scope: "scope1"}

	assert.True(t, m.AddHashes("h1", "h2"))
	assert.True(t, m.AddHashes("h2", "h3"))
	assert.False(t, m.AddHashes("h2", "h3"))

	assert.Len(t, m.GetHashes(), 3)
}

func TestGetPoint(t *testing.T) {
	db := prepare()

	m := NewMissionManager(db, nil)
	require.NoError(t, m.Migrate())

	m1 := &model.Mission{Name: "mission1", Scope: "scope1"}
	require.NoError(t, m.PutMission(m1))

	m.AddPoint(m1, newCotMessage("scope1", "uid1", 10, 20))

	di := m.GetPoint("uid1")

	require.NotNil(t, di)
	require.NotNil(t, di.GetEvent())
	assert.Equal(t, 10., di.GetEvent().GetLat())
}

func prepare() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
	if err != nil {
		panic("failed to connect database")
	}

	return db
}

func getSubscription(missionId uint, uid string) *model.Subscription {
	return &model.Subscription{
		MissionID:  missionId,
		ClientUID:  uid,
		Username:   "aaa",
		CreateTime: time.Now(),
		Role:       "aaa",
	}
}

func newCotMessage(scope, uid string, lat, lon float64) *cot.CotMessage {
	tak := cot.BasicMsg("a-f-G", uid, time.Second*10)
	tak.CotEvent.Lat = lat
	tak.CotEvent.Lon = lon

	det, _ := cot.DetailsFromString(tak.GetCotEvent().GetDetail().GetXmlDetail())

	return &cot.CotMessage{TakMessage: tak, Detail: det, Scope: scope}
}
