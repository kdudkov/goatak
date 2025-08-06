package database

import (
	"time"

	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/model"
)

type PointQuery struct {
	Query[model.Point]
	id    uint
	uid   string
	scope string
	stale *bool
}

func NewPointQuery(db *gorm.DB) *PointQuery {
	return &PointQuery{
		Query: Query[model.Point]{
			db:     db,
			limit:  100,
			offset: 0,
			order:  "created_at DESC",
		},
	}
}

func (q *PointQuery) Order(s string) *PointQuery {
	q.order = s
	return q
}

func (q *PointQuery) Limit(n int) *PointQuery {
	q.limit = n
	return q
}

func (q *PointQuery) Offset(n int) *PointQuery {

	q.offset = n
	return q
}

func (q *PointQuery) Id(id uint) *PointQuery {
	q.id = id
	return q
}

func (q *PointQuery) UID(uid string) *PointQuery {
	q.uid = uid
	return q
}

func (q *PointQuery) Scope(scope string) *PointQuery {
	q.scope = scope
	return q
}

func (q *PointQuery) Stale(b bool) *PointQuery {
	q.stale = &b
	return q
}

func (q *PointQuery) where() *gorm.DB {
	tx := q.db

	if q.id != 0 {
		tx = tx.Where("id = ?", q.id)
	}

	if q.uid != "" {
		tx = tx.Where("uid = ?", q.uid)
	}

	if q.stale != nil {
		if *q.stale {
			tx = tx.Where("stale < ?", time.Now())
		} else {
			tx = tx.Where("stale >= ?", time.Now())
		}
	}

	if q.scope != "" {
		tx = tx.Where("scope = ?", q.scope)
	}

	return tx
}

func (q *PointQuery) Get() []*model.Point {
	return q.get(q.where().Model(&model.Point{}))
}

func (q *PointQuery) One() *model.Point {
	return q.one(q.where().Model(&model.Point{}))
}

func (q *PointQuery) Update(updates map[string]any) error {
	return q.updateOrError(q.where().Model(&model.Point{}), updates)
}

func (q *PointQuery) Delete() error {
	return q.where().Delete(&model.Point{}).Error
}
