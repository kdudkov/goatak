package database

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/kdudkov/goatak/pkg/model"
)

func getTestDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&model.Device{}, &model.Certificate{})

	return db
}

func TestDeviceQuery_Count(t *testing.T) {
	db := getTestDatabase()

	db.Save(&model.Device{Login: "login1"})
	db.Save(&model.Device{Login: "login2"})

	db.Save(&model.Certificate{Serial: "1", Login: "login1", UID: "aaa"})
	db.Save(&model.Certificate{Serial: "2", Login: "login1", UID: "aaa"})
	db.Save(&model.Certificate{Serial: "3", Login: "login1", UID: "aaa"})
	db.Save(&model.Certificate{Serial: "4", Login: "login2", UID: "aaa"})

	res := NewDeviceQuery(db).Full().Get()

	require.Len(t, res, 2)
	require.True(t, len(res[0].Certs) > 0)
	require.True(t, len(res[1].Certs) > 0)
}
