package database

import (
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/model"
)

type DeviceQuery struct {
	Query[model.Device]
	login string
	scope string
	full  bool
}

func NewDeviceQuery(db *gorm.DB) *DeviceQuery {
	return &DeviceQuery{
		Query: Query[model.Device]{
			db:     db,
			limit:  100,
			offset: 0,
			order:  "login DESC",
		},
	}
}

func (q *DeviceQuery) Order(s string) *DeviceQuery {
	q.order = s
	return q
}

func (q *DeviceQuery) Limit(n int) *DeviceQuery {
	q.limit = n
	return q
}

func (q *DeviceQuery) Offset(n int) *DeviceQuery {

	q.offset = n
	return q
}

func (q *DeviceQuery) Login(login string) *DeviceQuery {
	q.login = login
	return q
}

func (q *DeviceQuery) Scope(scope string) *DeviceQuery {
	q.scope = scope
	return q
}

func (q *DeviceQuery) Full() *DeviceQuery {
	q.full = true
	return q
}

func (q *DeviceQuery) where() *gorm.DB {
	tx := q.db

	if q.login != "" {
		tx = tx.Where("login = ?", q.login)
	}

	if q.scope != "" {
		tx = tx.Where("scope = ?", q.scope)
	}

	if q.full {
		tx = tx.Preload("Certs", func(db *gorm.DB) *gorm.DB {
			return db.Order("certificates.last_connect desc")
		})
	}

	return tx
}

func (q *DeviceQuery) Get() []*model.Device {
	return q.get(q.where().Model(&model.Device{}))
}

func (q *DeviceQuery) One() *model.Device {
	return q.one(q.where().Model(&model.Device{}))
}

func (q *DeviceQuery) Count() int64 {
	return q.count(q.where().Model(&model.Device{}))
}

func (q *DeviceQuery) Update(updates map[string]any) error {
	return q.updateOrError(q.where().Model(&model.Device{}), updates)
}

func (q *DeviceQuery) Delete(login string) error {
	return q.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("login = ?", login).Delete(&model.Device{}).Error; err != nil {
			return err
		}

		return tx.Where("login = ?", login).Delete(&model.Certificate{}).Error
	})
}
