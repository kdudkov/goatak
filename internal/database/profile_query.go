package database

import (
	"time"

	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/model"
)

type ProfileQuery struct {
	Query[model.Profile]
	login string
	uid   string
	after time.Time
}

func NewProfileQuery(db *gorm.DB) *ProfileQuery {
	return &ProfileQuery{
		Query: Query[model.Profile]{
			db:     db,
			limit:  100,
			offset: 0,
			order:  "login,uid",
		},
	}
}

func (q *ProfileQuery) Order(s string) *ProfileQuery {
	q.order = s
	return q
}

func (q *ProfileQuery) Limit(n int) *ProfileQuery {
	q.limit = n
	return q
}

func (q *ProfileQuery) Offset(n int) *ProfileQuery {
	q.offset = n
	return q
}

func (q *ProfileQuery) Login(login string) *ProfileQuery {
	q.login = login
	return q
}

func (q *ProfileQuery) UID(uid string) *ProfileQuery {
	q.uid = uid
	return q
}

func (q *ProfileQuery) After(t time.Time) *ProfileQuery {
	q.after = t
	return q
}

func (q *ProfileQuery) where() *gorm.DB {
	tx := q.db

	if q.login != "" {
		tx = tx.Where("login = ?", q.login)
	}

	if q.uid != "" {
		tx = tx.Where("uid = ?", q.uid)
	}

	if !q.after.IsZero() {
		tx = tx.Where("update_time >= ?", q.after)
	}

	return tx
}

func (q *ProfileQuery) Get() []*model.Profile {
	return q.get(q.where().Model(&model.Profile{}))
}

func (q *ProfileQuery) One() *model.Profile {
	return q.one(q.where().Model(&model.Profile{}))
}

func (q *ProfileQuery) Count() int64 {
	return q.count(q.where().Model(&model.Profile{}))
}

func (q *ProfileQuery) Update(updates map[string]any) error {
	return q.updateOrError(q.where().Model(&model.Profile{}), updates)
}

func (q *ProfileQuery) Delete() error {
	return q.where().Delete(&model.Profile{}).Error
}
