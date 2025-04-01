package database

import (
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/model"
)

type ResourceQuery struct {
	Query[model.Resource]
	id    uint
	scope string
	tool  string
	hash  string
	uid   string
	name  string
}

func NewResourceQuery(db *gorm.DB) *ResourceQuery {
	return &ResourceQuery{
		Query: Query[model.Resource]{
			db:     db,
			limit:  100,
			offset: 0,
			order:  "resources.created_at",
		},
	}
}

func (q *ResourceQuery) Order(s string) *ResourceQuery {
	if q == nil {
		return nil
	}

	q.order = s
	return q
}

func (q *ResourceQuery) Limit(n int) *ResourceQuery {
	if q == nil {
		return nil
	}

	q.limit = n
	return q
}

func (q *ResourceQuery) Offset(n int) *ResourceQuery {
	if q == nil {
		return nil
	}

	q.offset = n
	return q
}

func (q *ResourceQuery) Id(id uint) *ResourceQuery {
	if q == nil {
		return nil
	}

	q.id = id
	return q
}

func (q *ResourceQuery) Scope(scope string) *ResourceQuery {
	if q == nil {
		return nil
	}

	q.scope = scope
	return q
}

func (q *ResourceQuery) UID(uid string) *ResourceQuery {
	if q == nil {
		return nil
	}

	q.uid = uid
	return q
}

func (q *ResourceQuery) Tool(tool string) *ResourceQuery {
	if q == nil {
		return nil
	}

	q.tool = tool
	return q
}

func (q *ResourceQuery) Hash(hash string) *ResourceQuery {
	if q == nil {
		return nil
	}

	q.hash = hash
	return q
}

func (q *ResourceQuery) Name(name string) *ResourceQuery {
	if q == nil {
		return nil
	}

	q.name = name
	return q
}

func (q *ResourceQuery) where() *gorm.DB {
	tx := q.db

	if q.id != 0 {
		tx = tx.Where("id = ?", q.id)
	}

	if q.scope != "" {
		tx = tx.Where("scope = ?", q.scope)
	}

	if q.hash != "" {
		tx = tx.Where("hash = ?", q.hash)
	}

	if q.uid != "" {
		tx = tx.Where("uid = ?", q.uid)
	}

	if q.tool != "" {
		tx = tx.Where("tool = ?", q.tool)
	}

	if q.name != "" {
		tx = tx.Where("name = ?", q.name)
	}

	return tx
}

func (q *ResourceQuery) Get() []*model.Resource {
	return q.get(q.where().Model(&model.Resource{}))
}

func (q *ResourceQuery) One() *model.Resource {
	return q.one(q.where().Model(&model.Resource{}))
}

func (q *ResourceQuery) Update(updates map[string]any) error {
	return q.updateOrError(q.where().Model(&model.Resource{}), updates)
}

func (q *ResourceQuery) Delete() error {
	return q.where().Delete(&model.Resource{}).Error
}
