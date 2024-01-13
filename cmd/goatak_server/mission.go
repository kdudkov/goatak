package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/cot"
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

	if err := mm.db.AutoMigrate(&model.DataItem{}); err != nil {
		return err
	}

	return nil
}

func (mm *MissionManager) GetAllMissions() []*model.Mission {
	if mm == nil || mm.db == nil {
		return nil
	}

	var result []*model.Mission

	mm.db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("timestamp desc")
	}).Find(&result)

	return result
}

func (mm *MissionManager) GetMission(name string) *model.Mission {
	var m *model.Mission

	result := mm.db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("timestamp desc")
	}).Take(&m, "name = ?", name)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	for _, p := range m.Items {
		_ = p.PostLoad()
	}

	return m
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

func (mm *MissionManager) DeleteMission(name string) {
	mm.db.Where("name = ?", name).Delete(&model.Mission{})
	mm.db.Where("mission_name = ?", name).Delete(&model.Subscription{})
	mm.db.Where("mission_name = ?", name).Delete(&model.Invitation{})
	mm.db.Where("mission_id = ?", name).Delete(&model.DataItem{})
}

func (mm *MissionManager) AddKw(name string, kw []string) {
	mm.db.Model(&model.Mission{}).Where("name = ?", name).Update("keywords", strings.Join(kw, ","))
}

func (mm *MissionManager) GetPoint(uid string) *model.DataItem {
	if mm == nil || mm.db == nil {
		return nil
	}

	var d *model.DataItem

	result := mm.db.Where("uid = ?", uid).Find(&d)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	return d
}

func (mm *MissionManager) AddPoint(name string, msg *cot.CotMessage) {
	if mm == nil || mm.db == nil {
		return
	}

	m := mm.GetMission(name)

	if m == nil {
		return
	}

	p, _ := msg.GetParent()

	for _, dp := range m.Items {
		if dp.UID == msg.GetUID() {
			dp.CreatorUID = p
			dp.Timestamp = msg.GetStartTime()
			dp.Type = msg.GetType()
			dp.Callsign = msg.GetCallsign()
			dp.IconsetPath = msg.GetIconsetPath()
			dp.Color = msg.GetColor()
			dp.Lat = msg.GetLat()
			dp.Lon = msg.GetLon()

			m.LastEdit = time.Now()
			mm.db.Save(m)
			return
		}
	}

	i := &model.DataItem{
		UID:         msg.GetUID(),
		CreatorUID:  p,
		Timestamp:   time.Now(),
		Type:        msg.GetType(),
		Callsign:    msg.GetCallsign(),
		Title:       "",
		IconsetPath: msg.GetIconsetPath(),
		Color:       msg.GetColor(),
		Lat:         msg.GetLat(),
		Lon:         msg.GetLon(),
		Event:       msg.TakMessage.GetCotEvent(),
	}

	m.Items = append(m.Items, i)
	m.LastEdit = time.Now()
	m.PreSave()

	mm.db.Save(m)
}

func (mm *MissionManager) DeletePoint(uid string) {
	if mm == nil || mm.db == nil {
		return
	}

	mm.db.Where("uid = ?", uid).Delete(&model.DataItem{})
}
func (mm *MissionManager) DeleteMissionPoints(missionId uint, uid string) {
	if mm == nil || mm.db == nil {
		return
	}

	mm.db.Where("mission_id = ? AND uid = ?", missionId, uid).Delete(&model.DataItem{})
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
	mm.db.Where("mission_name = ? AND client_uid = ?", name, uid).Delete(&model.Subscription{})
}

func (mm *MissionManager) GetInvitation(name string, uid string, typ string) *model.Invitation {
	var s *model.Invitation

	result := mm.db.Where("mission_name = ? AND client_uid = ? AND typ = ?", name, uid, typ).Take(&s)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	return s
}

func (mm *MissionManager) PutInvitation(s *model.Invitation) {
	old := mm.GetInvitation(s.MissionName, s.Invitee, s.Typ)

	if old != nil {
		mm.db.Delete(old, old.ID)
	}

	mm.db.Save(s)
}

func (mm *MissionManager) DeleteInvitation(name string, uid string, typ string) {
	mm.db.Where("mission_name = ? AND client_uid = ? AND typ = ?", name, uid, typ).Delete(&model.Invitation{})
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
