package main

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/kdudkov/goatak/internal/model"
)

type MissionManager struct {
	db *gorm.DB
}

func NewMissionManager(db *gorm.DB) *MissionManager {
	mn := &MissionManager{
		db: db,
	}

	return mn
}

func (mm *MissionManager) Migrate() error {
	if mm == nil || mm.db == nil {
		return fmt.Errorf("no database")
	}

	// Migrate the schema
	if err := mm.db.AutoMigrate(&model.Mission{}); err != nil {
		return err
	}

	if err := mm.db.AutoMigrate(&model.Subscription{}); err != nil {
		return err
	}

	if err := mm.db.AutoMigrate(&model.Invitation{}); err != nil {
		return err
	}

	return nil
}

func (mm *MissionManager) PutMission(m *model.Mission) error {
	if mm == nil || mm.db == nil {
		return nil
	}

	if m == nil {
		return fmt.Errorf("null mission")
	}

	if m.Name == "" {
		return fmt.Errorf("null mission name")
	}

	if mm.GetMission(m.Name) != nil {
		return fmt.Errorf("mission %s exists", m.Name)
	}

	tx := mm.db.Save(m)

	return tx.Error
}

func (mm *MissionManager) GetAll() []*model.Mission {
	if mm == nil || mm.db == nil {
		return nil
	}

	var result []*model.Mission
	mm.db.Find(&result)

	return result
}

func (mm *MissionManager) GetMission(name string) *model.Mission {
	var m *model.Mission

	result := mm.db.Where("name = ?", name).Take(&m)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	return m
}

func (mm *MissionManager) DeleteMission(name string) {
	mm.db.Where("name = ?", name).Delete(&model.Mission{})
	mm.db.Where("mission_name = ?", name).Delete(&model.Subscription{})
	mm.db.Where("mission_name = ?", name).Delete(&model.Invitation{})
}

func (mm *MissionManager) AddKw(name string, kw []string) {
	mm.db.Model(&model.Mission{}).Where("name = ?", name).Update("keywords", strings.Join(kw, ","))
}

func (mm *MissionManager) PutSubscription(s *model.Subscription) {
	old := mm.GetSubscription(s.MissionName, s.ClientUID)

	if old != nil {
		mm.db.Delete(old, old.ID)
	}

	mm.db.Save(s)
}

func (mm *MissionManager) GetSubscriptions(name string) []*model.Subscription {
	var s []*model.Subscription

	result := mm.db.Where("mission_name = ?", name).Find(&s)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	return s
}

func (mm *MissionManager) GetSubscribers(name string) []string {
	if mm == nil || mm.db == nil {
		return nil
	}

	var subscriptions []*model.Subscription

	result := mm.db.Where("mission_name = ?", name).Find(&subscriptions)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	res := make([]string, len(subscriptions))

	for i, s := range subscriptions {
		res[i] = s.ClientUID
	}

	return res
}

func (mm *MissionManager) GetSubscription(name string, uid string) *model.Subscription {
	var s *model.Subscription

	result := mm.db.Where("mission_name = ? AND client_uid = ?", name, uid).Take(&s)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	return s
}

func (mm *MissionManager) DeleteSubscription(name string, uid string) {
	if s := mm.GetSubscription(name, uid); s != nil {
		mm.db.Delete(s, s.ID)
	}
}

func (mm *MissionManager) PutInvitation(s *model.Invitation) {
	old := mm.GetInvitation(s.MissionName, s.ClientUID)

	if old != nil {
		mm.db.Delete(old, old.ID)
	}

	mm.db.Save(s)
}

func (mm *MissionManager) GetInvitation(name string, uid string) *model.Invitation {
	var s *model.Invitation

	result := mm.db.Where("mission_name = ? AND client_uid = ?", name, uid).Take(&s)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	return s
}

func (mm *MissionManager) GetInvitations(uid string) []string {
	var m []*model.Invitation

	result := mm.db.Where("client_uid = ?", uid).Find(&m)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	res := make([]string, len(m))

	for i, s := range m {
		res[i] = s.MissionName
	}

	return res
}
