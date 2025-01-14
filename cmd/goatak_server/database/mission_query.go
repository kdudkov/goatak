package database

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/kdudkov/goatak/internal/model"
)

type Query struct {
	db     *gorm.DB
	limit  int
	offset int
	order  string
}

type MissionQuery struct {
	Query
	id    uint
	name  string
	scope string
	full  bool
}

func NewMissionQuery(db *gorm.DB) *MissionQuery {
	return &MissionQuery{
		Query: Query{
			db:     db,
			limit:  100,
			offset: 0,
			order:  "missions.created_at",
		},
	}
}

func (q *MissionQuery) Order(s string) *MissionQuery {
	if q == nil {
		return nil
	}

	q.order = s
	return q
}

func (q *MissionQuery) Limit(n int) *MissionQuery {
	if q == nil {
		return nil
	}

	q.limit = n
	return q
}

func (q *MissionQuery) Offset(n int) *MissionQuery {
	if q == nil {
		return nil
	}

	q.offset = n
	return q
}

func (q *MissionQuery) Id(id uint) *MissionQuery {
	if q == nil {
		return nil
	}

	q.id = id
	return q
}

func (q *MissionQuery) Name(name string) *MissionQuery {
	if q == nil {
		return nil
	}

	q.name = name
	return q
}

func (q *MissionQuery) Scope(scope string) *MissionQuery {
	if q == nil {
		return nil
	}

	q.scope = scope
	return q
}

func (q *MissionQuery) Full() *MissionQuery {
	if q == nil {
		return nil
	}

	q.full = true
	return q
}

func (q *MissionQuery) where(tx *gorm.DB) *gorm.DB {
	if q.id != 0 {
		tx = tx.Where("missions.id = ?", q.id)
	}

	if q.name != "" {
		tx = tx.Where("missions.name = ?", q.name)
	}

	if q.scope != "" {
		tx = tx.Where("missions.scope = ?", q.scope)
	}

	return tx
}

func (q *MissionQuery) Get() []*model.Mission {
	if q == nil {
		return nil
	}

	var res []*model.Mission

	tx := q.where(q.db.Table("Missions"))

	if q.full {
		tx = tx.Preload("Points").Preload("Resources")
	}

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

func (q *MissionQuery) One() *model.Mission {
	if q == nil {
		return nil
	}

	res := new(model.Mission)

	tx := q.where(q.db.Table("missions"))

	if q.full {
		tx = tx.Preload("Points").Preload("Resources")
	}

	err := tx.Take(&res).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return res
}

func (q *MissionQuery) Update(updates map[string]any) error {
	if q == nil {
		return nil
	}

	res := q.where(q.db.Table("missions"))
	res.Updates(updates)

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return fmt.Errorf("Mission is not found")
	}

	return nil
}
