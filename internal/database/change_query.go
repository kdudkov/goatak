package database

import (
	"time"

	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/model"
)

type ChangeQuery struct {
	Query[model.Change]
	id        uint
	missionID uint
	after     time.Time
}

func NewChangeQuery(db *gorm.DB) *ChangeQuery {
	return &ChangeQuery{
		Query: Query[model.Change]{
			db:     db,
			limit:  100,
			offset: 0,
			order:  "changes.created_at DESC",
		},
	}
}

func (q *ChangeQuery) Order(s string) *ChangeQuery {
	if q == nil {
		return nil
	}

	q.order = s
	return q
}

func (q *ChangeQuery) Limit(n int) *ChangeQuery {
	if q == nil {
		return nil
	}

	q.limit = n
	return q
}

func (q *ChangeQuery) Offset(n int) *ChangeQuery {
	if q == nil {
		return nil
	}

	q.offset = n
	return q
}

func (q *ChangeQuery) Id(id uint) *ChangeQuery {
	if q == nil {
		return nil
	}

	q.id = id
	return q
}

func (q *ChangeQuery) Mission(id uint) *ChangeQuery {
	if q == nil {
		return nil
	}

	q.missionID = id
	return q
}

func (q *ChangeQuery) After(t time.Time) *ChangeQuery {
	if q == nil {
		return nil
	}

	q.after = t
	return q
}

func (q *ChangeQuery) where() *gorm.DB {
	tx := q.db.Joins("MissionPoint").Joins("Resource")

	if q.id != 0 {
		tx = tx.Where("id = ?", q.id)
	}

	if q.missionID != 0 {
		tx = tx.Where("changes.mission_id = ?", q.missionID)
	}

	if !q.after.IsZero() {
		tx = tx.Where("changes.created_at > ?", q.after)
	}

	return tx
}

func (q *ChangeQuery) Get() []*model.Change {
	return q.get(q.where().Model(&model.Change{}))
}

func (q *ChangeQuery) One() *model.Change {
	return q.one(q.where().Model(&model.Change{}))
}

func (q *ChangeQuery) Update(updates map[string]any) error {
	return q.updateOrError(q.where().Model(&model.Change{}), updates)
}

func (q *ChangeQuery) Delete() error {
	return q.where().Delete(&model.Change{}).Error
}
