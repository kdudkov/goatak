package database

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/kdudkov/goatak/internal/model"
)

func (mm *DatabaseManager) CreateMission(m *model.Mission) error {
	if mm == nil || mm.db == nil {
		return nil
	}

	if m == nil {
		return fmt.Errorf("null mission")
	}

	if m.Name == "" {
		return fmt.Errorf("null mission name")
	}

	return mm.db.Transaction(func(tx *gorm.DB) error {
		if NewMissionQuery(tx).Scope(m.Scope).Name(m.Name).One() != nil {
			return fmt.Errorf("mission %s exists", m.Name)
		}

		if err := tx.Create(m).Error; err != nil {
			return err
		}

		c := &model.Change{
			CreatedAt:  time.Now(),
			Type:       "CREATE_MISSION",
			MissionID:  m.ID,
			CreatorUID: m.CreatorUID,
		}

		return tx.Create(c).Error
	})
}

func (mm *DatabaseManager) DeleteMission(id uint) error {
	return mm.db.Transaction(func(tx *gorm.DB) error {
		tables := []any{
			&model.Subscription{},
			&model.Invitation{},
			&model.Change{},
			&model.MissionPoint{},
			&model.MissionFile{}}

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

func (mm *DatabaseManager) AddKw(name string, kw []string) error {
	return mm.MissionQuery().Name(name).Update(map[string]any{"keywords": strings.Join(kw, ",")})
}
