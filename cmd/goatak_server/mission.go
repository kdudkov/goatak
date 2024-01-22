package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/cot"
)

type MissionManager struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

func NewMissionManager(db *gorm.DB, logger *zap.SugaredLogger) *MissionManager {
	mn := &MissionManager{
		db:     db,
		logger: logger,
	}

	return mn
}

func (mm *MissionManager) Save(s any) {
	if mm == nil || mm.db == nil {
		return
	}

	mm.db.Save(s)
}

func (mm *MissionManager) Migrate() error {
	if mm == nil || mm.db == nil {
		return fmt.Errorf("no database")
	}

	// Migrate the schema
	if err := mm.db.AutoMigrate(
		&model.Mission{},
		&model.Subscription{},
		&model.Invitation{},
		&model.DataItem{},
		&model.Change{}); err != nil {
		return err
	}

	return nil
}

func (mm *MissionManager) GetAllMissionsAdm() []*model.Mission {
	if mm == nil || mm.db == nil {
		return nil
	}

	var result []*model.Mission

	mm.db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("timestamp desc")
	}).Find(&result)

	return result
}

func (mm *MissionManager) GetAllMissions(scope string) []*model.Mission {
	if mm == nil || mm.db == nil {
		return nil
	}

	var result []*model.Mission

	mm.db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("timestamp desc")
	}).Where("scope = ?", scope).Find(&result)

	return result
}

func (mm *MissionManager) GetMissionById(id uint) *model.Mission {
	var m *model.Mission

	result := mm.db.Preload("Items").Take(&m, "id = ?", id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	return m
}

func (mm *MissionManager) GetMission(scope, name string) *model.Mission {
	var m *model.Mission

	result := mm.db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("timestamp desc")
	}).Take(&m, "scope = ? and name = ?", scope, name)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
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

	if mm.GetMission(m.Scope, m.Name) != nil {
		return fmt.Errorf("mission %s exists", m.Name)
	}

	tx := mm.db.Create(m)

	if tx.Error != nil {
		return tx.Error
	}

	c := &model.Change{
		CreateTime: time.Now(),
		Type:       "CREATE_MISSION",
		MissionID:  m.ID,
		CreatorUID: m.CreatorUID,
	}

	tx = mm.db.Create(c)

	return tx.Error
}

func (mm *MissionManager) DeleteMission(id uint) {
	mm.db.Where("id = ?", id).Delete(&model.Mission{})
	mm.db.Where("mission_id = ?", id).Delete(&model.Subscription{})
	mm.db.Where("mission_id = ?", id).Delete(&model.Invitation{})
	mm.db.Where("mission_id = ?", id).Delete(&model.DataItem{})
	mm.db.Where("mission_id = ?", id).Delete(&model.Change{})
}

func (mm *MissionManager) AddKw(name string, kw []string) {
	mm.db.Model(&model.Mission{}).Where("name = ?", name).Update("keywords", strings.Join(kw, ","))
}

func (mm *MissionManager) GetPoint(uid string) *model.DataItem {
	if mm == nil || mm.db == nil {
		return nil
	}

	var d *model.DataItem

	err := mm.db.Where("uid = ?", uid).Find(&d).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return d
}

func (mm *MissionManager) AddPoint(name string, msg *cot.CotMessage) {
	if mm == nil || mm.db == nil {
		return
	}

	m := mm.GetMission(msg.Scope, name)

	if m == nil {
		return
	}

	now := time.Now()

	for _, dp := range m.Items {
		if dp.UID == msg.GetUID() {
			dp.UpdateFromMsg(msg)
			mm.db.Save(dp)

			return
		}
	}

	i := &model.DataItem{
		UID: msg.GetUID(),
	}

	i.UpdateFromMsg(msg)

	m.Items = append(m.Items, i)
	m.LastEdit = now

	mm.db.Save(m)

	p, _ := msg.GetParent()

	c := &model.Change{
		CreateTime:  now,
		Type:        "ADD_CONTENT",
		MissionID:   m.ID,
		CreatorUID:  p,
		ContentUID:  msg.GetUID(),
		CotType:     msg.GetType(),
		Callsign:    msg.GetCallsign(),
		IconsetPath: msg.GetIconsetPath(),
		Color:       msg.GetColor(),
		Lat:         msg.GetLat(),
		Lon:         msg.GetLon(),
	}

	mm.db.Create(c)
}

func (mm *MissionManager) DeletePoint(uid string) {
	if mm == nil || mm.db == nil {
		return
	}

	mm.db.Where("uid = ?", uid).Delete(&model.DataItem{})
}

func (mm *MissionManager) DeleteMissionPoints(missionId uint, uid string) {
	if mm == nil || mm.db == nil || uid == "" {
		return
	}

	var mp *model.DataItem

	err := mm.db.Where("mission_id = ? AND uid = ?", missionId, uid).Find(&mp).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}

	err = mm.db.Where("id = ?", mp.ID).Delete(&model.DataItem{}).Error

	if err != nil {
		mm.logger.Errorf("error %v", err)
		return
	}

	if mp == nil {
		mm.logger.Errorf("mp is nil")
		return
	}

	c := &model.Change{
		CreateTime:  time.Now(),
		Type:        "REMOVE_CONTENT",
		MissionID:   missionId,
		CreatorUID:  "",
		ContentUID:  mp.UID,
		CotType:     mp.Type,
		Callsign:    mp.Callsign,
		IconsetPath: mp.IconsetPath,
		Color:       mp.Color,
		Lat:         mp.Lat,
		Lon:         mp.Lon,
	}

	mm.db.Create(c)
}

func (mm *MissionManager) DeleteMissionContent(missionId uint, hash string) {
	if mm == nil || mm.db == nil || hash == "" {
		return
	}

	m := mm.GetMissionById(missionId)

	if m == nil {
		return
	}

	if m.RemoveHash(hash) {
		m.LastEdit = time.Now()
		mm.db.Save(m)
	}
}

func (mm *MissionManager) PutSubscription(s *model.Subscription) {
	old := mm.GetSubscription(s.MissionID, s.ClientUID)

	if old != nil {
		mm.db.Delete(old, old.ID)
	}

	mm.db.Save(s)
}

func (mm *MissionManager) GetSubscriptions(missionId uint) []*model.Subscription {
	var s []*model.Subscription

	result := mm.db.Where("mission_id = ?", missionId).Find(&s)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	return s
}

func (mm *MissionManager) GetSubscribers(missionId uint) []string {
	if mm == nil || mm.db == nil {
		return nil
	}

	var subscriptions []*model.Subscription

	result := mm.db.Where("mission_id = ?", missionId).Find(&subscriptions)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	res := make([]string, len(subscriptions))

	for i, s := range subscriptions {
		res[i] = s.ClientUID
	}

	return res
}

func (mm *MissionManager) GetSubscription(missionId uint, uid string) *model.Subscription {
	var s *model.Subscription

	result := mm.db.Where("mission_id = ? AND client_uid = ?", missionId, uid).Take(&s)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	return s
}

func (mm *MissionManager) DeleteSubscription(name string, uid string) {
	mm.db.Where("mission_name = ? AND client_uid = ?", name, uid).Delete(&model.Subscription{})
}

func (mm *MissionManager) GetInvitation(missionId uint, uid string, typ string) *model.Invitation {
	var s *model.Invitation

	result := mm.db.Where("mission_id = ? AND client_uid = ? AND typ = ?", missionId, uid, typ).Take(&s)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	return s
}

func (mm *MissionManager) PutInvitation(s *model.Invitation) {
	old := mm.GetInvitation(s.MissionID, s.Invitee, s.Typ)

	if old != nil {
		mm.db.Delete(old, old.ID)
	}

	mm.db.Save(s)
}

func (mm *MissionManager) DeleteInvitation(missionId uint, uid string, typ string) {
	mm.db.Where("mission_id = ? AND client_uid = ? AND typ = ?", missionId, uid, typ).Delete(&model.Invitation{})
}

func (mm *MissionManager) GetInvitations(uid string) []string {
	var m []*model.Invitation

	result := mm.db.Where("client_uid = ?", uid).Find(&m)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	res := make([]string, len(m))

	for i, s := range m {
		mission := mm.GetMissionById(s.MissionID)
		res[i] = mission.Name
	}

	return res
}

func (mm *MissionManager) GetChanges(missionId uint, after time.Time) []*model.Change {
	var m []*model.Change

	err := mm.db.Where("mission_id = ? and create_time > ?", missionId, after).Order("create_time DESC").Find(&m).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return m
}
