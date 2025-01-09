package database

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/kdudkov/goatak/internal/model"
)

type FileQuery struct {
	Query
	id    uint
	scope string
	tool  string
	hash  string
	uid   string
	name  string
}

func NewFileQuery(db *gorm.DB) *FileQuery {
	return &FileQuery{
		Query: Query{
			db:     db,
			limit:  100,
			offset: 0,
			order:  "contents.created_at",
		},
	}
}

func (q *FileQuery) Order(s string) *FileQuery {
	if q == nil {
		return nil
	}

	q.order = s
	return q
}

func (q *FileQuery) Limit(n int) *FileQuery {
	if q == nil {
		return nil
	}

	q.limit = n
	return q
}

func (q *FileQuery) Offset(n int) *FileQuery {
	if q == nil {
		return nil
	}

	q.offset = n
	return q
}

func (q *FileQuery) Id(id uint) *FileQuery {
	if q == nil {
		return nil
	}

	q.id = id
	return q
}

func (q *FileQuery) Scope(scope string) *FileQuery {
	if q == nil {
		return nil
	}

	q.scope = scope
	return q
}

func (q *FileQuery) UID(uid string) *FileQuery {
	if q == nil {
		return nil
	}

	q.uid = uid
	return q
}

func (q *FileQuery) Tool(tool string) *FileQuery {
	if q == nil {
		return nil
	}

	q.tool = tool
	return q
}

func (q *FileQuery) Hash(hash string) *FileQuery {
	if q == nil {
		return nil
	}

	q.hash = hash
	return q
}

func (q *FileQuery) Name(name string) *FileQuery {
	if q == nil {
		return nil
	}

	q.name = name
	return q
}

func (q *FileQuery) where(tx *gorm.DB) *gorm.DB {
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

func (q *FileQuery) Get() []*model.Content {
	if q == nil {
		return nil
	}

	var res []*model.Content

	tx := q.where(q.db.Table("contents"))

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

func (q *FileQuery) One() *model.Content {
	if q == nil {
		return nil
	}

	res := new(model.Content)

	tx := q.where(q.db.Table("contents"))

	err := tx.Take(&res).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return res
}

func (q *FileQuery) Update(updates map[string]any) error {
	if q == nil {
		return nil
	}

	res := q.where(q.db.Table("contents"))
	res.Updates(updates)

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return fmt.Errorf("Content is not found")
	}

	return nil
}
