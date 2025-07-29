package database

import (
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/model"
	"github.com/kdudkov/goatak/pkg/util"
)

type FeedQuery struct {
	Query[model.Feed2]
	uid   string
	user  string
	scope util.StringSet
	all   bool
}

func NewFeedQuery(db *gorm.DB) *FeedQuery {
	return &FeedQuery{
		Query: Query[model.Feed2]{
			db:     db,
			limit:  100,
			offset: 0,
			order:  "uid",
		},
		scope: util.NewStringSet(),
	}
}

func (q *FeedQuery) Order(s string) *FeedQuery {
	q.order = s
	return q
}

func (q *FeedQuery) Limit(n int) *FeedQuery {
	q.limit = n
	return q
}

func (q *FeedQuery) Offset(n int) *FeedQuery {
	q.offset = n
	return q
}

func (q *FeedQuery) UID(uid string) *FeedQuery {
	q.uid = uid
	return q
}

func (q *FeedQuery) User(user string) *FeedQuery {
	q.user = user
	return q
}

func (q *FeedQuery) Scope(scope string) *FeedQuery {
	if q == nil {
		return nil
	}

	q.scope.Add(scope)

	return q
}

func (q *FeedQuery) ReadScope(scope []string) *FeedQuery {
	if q == nil {
		return nil
	}

	q.scope.Add(scope...)

	return q
}

func (q *FeedQuery) All(all bool) *FeedQuery {
	q.all = all
	return q
}

func (q *FeedQuery) where() *gorm.DB {
	tx := q.db

	if q.uid != "" {
		tx = tx.Where("uid = ?", q.uid)
	}

	if q.user != "" {
		tx = tx.Where("user = ?", q.user)
	}

	if len(q.scope) > 0 && !q.scope.Has("*") {
		tx = tx.Where("scope in (?)", q.scope.List())
	}

	if !q.all {
		tx = tx.Where("active is true")
	}

	return tx
}

func (q *FeedQuery) Get() []*model.Feed2 {
	return q.get(q.where().Model(&model.Feed2{}))
}

func (q *FeedQuery) One() *model.Feed2 {
	return q.one(q.where().Model(&model.Feed2{}))
}

func (q *FeedQuery) Count() int64 {
	return q.count(q.where().Model(&model.Feed2{}))
}

func (q *FeedQuery) Update(updates map[string]any) error {
	return q.updateOrError(q.where().Model(&model.Feed2{}), updates)
}

func (q *FeedQuery) Delete() error {
	return q.where().Delete(&model.Feed2{}).Error
}
