package main

import (
	"os"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/cot"
)

const db_name = "test.db"

func TestMissionSubscriptions(t *testing.T) {
	db := prepare()
	defer rmDatabase()

	m := NewMissionManager(db)
	require.NoError(t, m.Migrate())

	m.PutSubscription(getSubscription("m1", "uid1"))
	m.PutSubscription(getSubscription("m1", "uid1"))
	m.PutSubscription(getSubscription("m1", "uid2"))
	m.PutSubscription(getSubscription("m2", "uid1"))

	assert.Len(t, m.GetSubscriptions("m1"), 2)
	assert.Len(t, m.GetSubscribers("m1"), 2)
	assert.Len(t, m.GetSubscriptions("m2"), 1)
	assert.Len(t, m.GetSubscribers("m2"), 1)

	s1 := m.GetSubscription("m1", "uid1")
	assert.Equal(t, "m1", s1.MissionName)
	assert.Equal(t, "aaa", s1.RoleType)
}

func TestMissionCRUD(t *testing.T) {
	db := prepare()
	defer rmDatabase()

	m := NewMissionManager(db)
	require.NoError(t, m.Migrate())

	require.NoError(t, m.PutMission(&model.Mission{Name: "m1"}))
	require.NoError(t, m.PutMission(&model.Mission{Name: "m2"}))
	require.Error(t, m.PutMission(&model.Mission{Name: "m2"}))

	assert.Len(t, m.GetAll(), 2)

	m.PutSubscription(getSubscription("m1", "uid1"))
	m.PutSubscription(getSubscription("m1", "uid1"))
	m.PutSubscription(getSubscription("m1", "uid2"))
	m.PutSubscription(getSubscription("m2", "uid1"))

	assert.Len(t, m.GetSubscriptions("m1"), 2)
	assert.Len(t, m.GetSubscribers("m1"), 2)
	assert.Len(t, m.GetSubscriptions("m2"), 1)
	assert.Len(t, m.GetSubscribers("m2"), 1)

	m.DeleteMission("m2")
	assert.Len(t, m.GetAll(), 1)

	assert.Len(t, m.GetSubscriptions("m1"), 2)
	assert.Len(t, m.GetSubscribers("m1"), 2)
	assert.Empty(t, m.GetSubscriptions("m2"))
	assert.Empty(t, m.GetSubscribers("m2"))
}

func TestAddPoint(t *testing.T) {
	db := prepare()
	defer rmDatabase()

	m := NewMissionManager(db)
	require.NoError(t, m.Migrate())

	require.NoError(t, m.PutMission(&model.Mission{Name: "m1"}))
	require.NoError(t, m.PutMission(&model.Mission{Name: "m2"}))

	m.AddPoint("m1", newCotMessage("uid1", 10, 20))
	m.AddPoint("m1", newCotMessage("uid2", 10, 20))
	m.AddPoint("m1", newCotMessage("uid1", 15, 20))
	m.AddPoint("m2", newCotMessage("uid1", 15, 20))

	assert.Len(t, m.GetMission("m1").Items, 2)
	assert.Len(t, m.GetMission("m2").Items, 1)

	m.DeletePoint("uid1")

	assert.Len(t, m.GetMission("m1").Items, 1)
	assert.Empty(t, m.GetMission("m2").Items)
}

func prepare() *gorm.DB {
	rmDatabase()

	db, err := gorm.Open(sqlite.Open(db_name), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	return db
}

func rmDatabase() {
	_ = os.Remove(db_name)
}

func getSubscription(name, uid string) *model.Subscription {
	return &model.Subscription{
		MissionName: name,
		ClientUID:   uid,
		Username:    "aaa",
		CreateTime:  time.Now(),
		RoleType:    "aaa",
		Permissions: "aaa",
	}
}

func newCotMessage(uid string, lat, lon float64) *cot.CotMessage {
	tak := cot.BasicMsg("a-f-G", uid, time.Second*10)
	tak.CotEvent.Lat = lat
	tak.CotEvent.Lon = lon

	det, _ := cot.DetailsFromString(tak.GetCotEvent().GetDetail().GetXmlDetail())

	return &cot.CotMessage{TakMessage: tak, Detail: det}
}
