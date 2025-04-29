package database

import (
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/model"
	"github.com/kdudkov/goatak/pkg/util"
)

type MissionQuery struct {
	Query[model.Mission]
	id    uint
	name  string
	scope util.StringSet
	full  bool
}

func NewMissionQuery(db *gorm.DB) *MissionQuery {
	return &MissionQuery{
		Query: Query[model.Mission]{
			db:     db,
			limit:  100,
			offset: 0,
			order:  "missions.created_at",
		},
		scope: util.NewStringSet(),
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

	q.scope.Add(scope)

	return q
}

func (q *MissionQuery) ReadScope(scope []string) *MissionQuery {
	if q == nil {
		return nil
	}

	q.scope.Add(scope...)

	return q
}

func (q *MissionQuery) Full() *MissionQuery {
	if q == nil {
		return nil
	}

	q.full = true
	return q
}

func (q *MissionQuery) where() *gorm.DB {
	tx := q.db

	if q.id != 0 {
		tx = tx.Where("missions.id = ?", q.id)
	}

	if q.name != "" {
		tx = tx.Where("missions.name = ?", q.name)
	}

	if len(q.scope) > 0 && !q.scope.Has("*") {
		tx = tx.Where("missions.scope in (?)", q.scope.List())
	}

	if q.full {
		tx = tx.Preload("Points").Preload("Resources")
	}

	return tx
}

func (q *MissionQuery) Get() []*model.Mission {
	return q.get(q.where().Model(&model.Mission{}))
}

func (q *MissionQuery) One() *model.Mission {
	return q.one(q.where().Model(&model.Mission{}))
}

func (q *MissionQuery) Update(updates map[string]any) error {
	return q.updateOrError(q.where().Model(&model.Mission{}), updates)
}

func (q *MissionQuery) Delete(id uint) error {
	return q.db.Transaction(func(tx *gorm.DB) error {
		tables := []any{
			&model.Subscription{},
			&model.Invitation{},
			&model.Change{},
		}

		if err := tx.Where("id = ?", id).Delete(&model.Mission{}).Error; err != nil {
			return err
		}

		for _, n := range tables {
			if err := tx.Where("mission_id = ?", id).Delete(n).Error; err != nil {
				return err
			}
		}

		return nil
	})

}

func has(arr []string, s string) bool {
	for _, s1 := range arr {
		if s1 == s {
			return true
		}
	}

	return false
}
