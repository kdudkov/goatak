package database

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/kdudkov/goatak/internal/model"
)

type ResourceQuery struct {
	Query
	id    uint
	scope string
	tool  string
	hash  string
	uid   string
	name  string
}

func NewResourceQuery(db *gorm.DB) *ResourceQuery {
	return &ResourceQuery{
		Query: Query{
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

func (q *ResourceQuery) where(tx *gorm.DB) *gorm.DB {
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
	if q == nil {
		return nil
	}

	var res []*model.Resource

	tx := q.where(q.db.Table("resources"))

	if q.order != "" {
		tx = tx.Order(q.order)
	}

	if q.limit > 0 {
		tx = tx.Limit(q.limit)
	}

	if q.offset > 0 {
		tx = tx.Offset(q.offset)
	}

	err := tx.Find(&res).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return res
}

func (q *ResourceQuery) One() *model.Resource {
	if q == nil {
		return nil
	}

	res := new(model.Resource)

	tx := q.where(q.db.Table("resources"))

	err := tx.Take(&res).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return res
}

func (q *ResourceQuery) Update(updates map[string]any) error {
	if q == nil {
		return nil
	}

	res := q.where(q.db.Table("resources"))
	res.Updates(updates)

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return fmt.Errorf("Resource is not found")
	}

	return nil
}
