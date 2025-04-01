package database

import (
	"log/slog"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func GetDatabase(dsn string, debug bool) (*gorm.DB, error) {
	conf := &gorm.Config{}

	if !debug {
		conf.Logger = logger.Default.LogMode(logger.Silent)
	} else {
		conf.Logger = logger.Default.LogMode(logger.Info)
	}

	var db *gorm.DB
	var err error

	if strings.HasPrefix(dsn, "mysql:") {
		slog.Info("open mysql database")
		db, err = gorm.Open(mysql.Open(strings.TrimPrefix(dsn, "mysql:")), conf)
	} else {
		slog.Info("open mysql database " + dsn)
		db, err = gorm.Open(sqlite.Open(dsn), conf)
	}

	if err != nil {
		slog.Error("db open error", slog.Any("error", err))
		return nil, err
	}

	return db, nil
}
