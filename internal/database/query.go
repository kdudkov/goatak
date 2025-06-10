package database

import (
	"errors"
	"log/slog"

	"gorm.io/gorm"
)

var errUpdate = errors.New("no record found")

type Query[T any] struct {
	db     *gorm.DB
	limit  int
	offset int
	order  string
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
	} else {
		if err != nil {
			slog.Error("db get error", slog.Any("error", err))
		}
	}

	return res
}

func (q *Query[T]) one(tx *gorm.DB) *T {
	res := new(T)

	err := tx.Take(&res).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else {
		if err != nil {
			slog.Error("db get one error", slog.Any("error", err))
		}
	}

	return res
}

func (q *Query[T]) count(tx *gorm.DB) int64 {
	var count int64

	err := tx.Count(&count).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.Error("count error", slog.Any("error", err))
	}

	return count
}

func (q *Query[T]) update(tx *gorm.DB, updates map[string]any) (int64, error) {
	tx.Updates(updates)

	if err := tx.Error; err != nil {
		slog.Error("update error", slog.Any("error", err))
		return 0, err
	}

	return tx.RowsAffected, nil
}

func (q *Query[T]) updateOrError(tx *gorm.DB, updates map[string]any) error {
	tx.Updates(updates)

	if err := tx.Error; err != nil {
		slog.Error("update error", slog.Any("error", err))
		return err
	}

	if tx.RowsAffected == 0 {
		return errUpdate
	}

	return nil
}
