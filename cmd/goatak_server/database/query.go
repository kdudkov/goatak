package database

import (
	"errors"

	"gorm.io/gorm"
)

var errUpdate = errors.New("no record found")

type Query[T any] struct {
	db     *gorm.DB
	limit  int
	offset int
	order  string
}

func (q *Query[T]) setDefaults(db *gorm.DB, order string) {
	q.db = db
	q.offset = 0
	q.limit = 100
	q.order = order
}

func (q *Query[T]) get(tx *gorm.DB) []*T {
	var res []*T

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

func (q *Query[T]) one(tx *gorm.DB) *T {
	res := new(T)

	err := tx.Take(&res).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return res
}

func (q *Query[T]) update(tx *gorm.DB, updates map[string]any) (int64, error) {
	tx.Updates(updates)

	if tx.Error != nil {
		return 0, tx.Error
	}

	return tx.RowsAffected, nil
}

func (q *Query[T]) updateOrError(tx *gorm.DB, updates map[string]any) error {
	tx.Updates(updates)

	if tx.Error != nil {
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return errUpdate
	}

	return nil
}
