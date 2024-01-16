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

	m := NewMissionManager(db)
	require.NoError(t, m.Migrate())

	m1 := &model.Mission{Name: "mission1"}
	m2 := &model.Mission{Name: "mission2"}

	require.NoError(t, m.PutMission(m1))
	require.NoError(t, m.PutMission(m2))

	require.NotEmpty(t, m1.ID)
	require.NotEmpty(t, m2.ID)

	m.PutSubscription(getSubscription(m1.ID, "uid1"))
	m.PutSubscription(getSubscription(m1.ID, "uid1"))
	m.PutSubscription(getSubscription(m1.ID, "uid2"))
	m.PutSubscription(getSubscription(m2.ID, "uid1"))

	assert.Len(t, m.GetSubscriptions(m1.ID), 2)
	assert.Len(t, m.GetSubscribers(m1.Name), 2)
	assert.Len(t, m.GetSubscriptions(m2.ID), 1)
	assert.Len(t, m.GetSubscribers(m2.Name), 1)

	s1 := m.GetSubscription(m1.ID, "uid1")
	assert.Equal(t, m1.ID, s1.MissionID)
	assert.Equal(t, "aaa", s1.Role)
}

func TestMissionCRUD(t *testing.T) {
	db := prepare()

	m := NewMissionManager(db)
	require.NoError(t, m.Migrate())

	m1 := &model.Mission{Name: "mission1"}
	m2 := &model.Mission{Name: "mission2"}

	require.NoError(t, m.PutMission(m1))
	require.NoError(t, m.PutMission(m2))

	require.Error(t, m.PutMission(&model.Mission{Name: "mission2"}))

	assert.Len(t, m.GetAllMissions(), 2)

	m.PutSubscription(getSubscription(m1.ID, "uid1"))
	m.PutSubscription(getSubscription(m1.ID, "uid1"))
	m.PutSubscription(getSubscription(m1.ID, "uid2"))
	m.PutSubscription(getSubscription(m2.ID, "uid1"))

	assert.Len(t, m.GetSubscriptions(m1.ID), 2)
	assert.Len(t, m.GetSubscribers(m1.Name), 2)
	assert.Len(t, m.GetSubscriptions(m2.ID), 1)
	assert.Len(t, m.GetSubscribers(m2.Name), 1)

	m.DeleteMission(m2.ID)
	assert.Len(t, m.GetAllMissions(), 1)

	assert.Len(t, m.GetSubscriptions(m1.ID), 2)
	assert.Len(t, m.GetSubscribers(m1.Name), 2)
	assert.Empty(t, m.GetSubscriptions(m2.ID))
	assert.Empty(t, m.GetSubscribers(m2.Name))
}

func TestAddPoint(t *testing.T) {
	db := prepare()

	m := NewMissionManager(db)
	require.NoError(t, m.Migrate())

	m1 := &model.Mission{Name: "mission1"}
	m2 := &model.Mission{Name: "mission2"}

	require.NoError(t, m.PutMission(m1))
	require.NoError(t, m.PutMission(m2))

	m.AddPoint(m1.Name, newCotMessage("uid1", 10, 20))
	m.AddPoint(m1.Name, newCotMessage("uid2", 10, 20))
	m.AddPoint(m1.Name, newCotMessage("uid1", 15, 20))
	m.AddPoint(m2.Name, newCotMessage("uid1", 15, 20))

	assert.Len(t, m.GetMission(m1.Name).Items, 2)
	assert.Len(t, m.GetMission(m2.Name).Items, 1)

	m.DeletePoint("uid1")

	assert.Len(t, m.GetMission(m1.Name).Items, 1)
	assert.Empty(t, m.GetMission(m2.Name).Items)
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

func newCotMessage(uid string, lat, lon float64) *cot.CotMessage {
	tak := cot.BasicMsg("a-f-G", uid, time.Second*10)
	tak.CotEvent.Lat = lat
	tak.CotEvent.Lon = lon

	det, _ := cot.DetailsFromString(tak.GetCotEvent().GetDetail().GetXmlDetail())

	return &cot.CotMessage{TakMessage: tak, Detail: det}
}
