package database

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/cot"
)

type DatabaseManager struct {
	db     *gorm.DB
	logger *slog.Logger
}

func New(db *gorm.DB) *DatabaseManager {
	mn := &DatabaseManager{
		db:     db,
		logger: slog.With("logger", "dbm"),
	}

	return mn
}

func (mm *DatabaseManager) Create(s any) error {
	if mm == nil || mm.db == nil {
		return nil
	}

	err := mm.db.Create(s).Error

	if err != nil {
		mm.logger.Error("error create object", slog.Any("error", err))
	}

	return err
}

func (mm *DatabaseManager) Save(s any) error {
	if mm == nil || mm.db == nil {
		return nil
	}

	err := mm.db.Save(s).Error

	if err != nil {
		mm.logger.Error("error saving object", slog.Any("error", err))
	}

	return err
}

func (mm *DatabaseManager) MissionQuery() *MissionQuery {
	if mm == nil || mm.db == nil {
		return nil
	}

	return NewMissionQuery(mm.db)
}

func (mm *DatabaseManager) FileQuery() *FileQuery {
	if mm == nil || mm.db == nil {
		return nil
	}

	return NewFileQuery(mm.db)
}

func (mm *DatabaseManager) Migrate() error {
	if mm == nil || mm.db == nil {
		return fmt.Errorf("no database")
	}

	// Migrate the schema
	if err := mm.db.AutoMigrate(
		&model.Mission{},
		&model.Change{},
		&model.MissionPoint{},
		&model.MissionFile{},
		&model.Subscription{},
		&model.Invitation{},
		&model.Content{}); err != nil {
		return err
	}

	return nil
}

func (mm *DatabaseManager) GetPoint(uid string) *model.MissionPoint {
	if mm == nil || mm.db == nil {
		return nil
	}

	var d *model.MissionPoint

	err := mm.db.Where("uid = ?", uid).Take(&d).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return d
}

func (mm *DatabaseManager) UpdateMissionChanged(id uint) {
	mm.db.Table("missions").Where("id = ?", id).Update("updated_at", time.Now())
}

func (mm *DatabaseManager) UpdateContentTool(id uint, tool string) {
	mm.db.Table("contents").Where("id = ?", id).Update("tool", tool)
}

func (mm *DatabaseManager) AddMissionPoint(mission *model.Mission, msg *cot.CotMessage) *model.Change {
	if mission == nil {
		return nil
	}

	now := time.Now()

	var point *model.MissionPoint
	for _, p := range mission.Points {
		if p.UID == msg.GetUID() {
			mm.logger.Info("update existing point " + p.Callsign + " " + p.UID)
			p.UpdateFromMsg(msg)
			if err := mm.Save(p); err != nil {
				return nil
			}
			point = p
			break
		}
	}

	if point == nil {
		mm.logger.Info("add new point " + msg.GetCallsign() + " " + msg.GetUID())
		point = &model.MissionPoint{
			UID: msg.GetUID(),
		}

		point.UpdateFromMsg(msg)

		mission.Points = append(mission.Points, point)
		mission.UpdatedAt = now

		if err := mm.Save(mission); err != nil {
			return nil
		}
	}

	// todo: use sender uid, not parent
	parent, _ := msg.GetParent()

	c := &model.Change{
		Type:        "ADD_CONTENT",
		MissionID:   mission.ID,
		CreatorUID:  parent,
		ContentUID:  msg.GetUID(),
		CotType:     msg.GetType(),
		Callsign:    msg.GetCallsign(),
		IconsetPath: msg.GetIconsetPath(),
		Color:       msg.GetColor(),
		Lat:         msg.GetLat(),
		Lon:         msg.GetLon(),
	}

	_ = mm.Create(c)

	return c
}

func (mm *DatabaseManager) DeleteMissionPoint(missionId uint, uid string, authorUID string) *model.Change {
	if mm == nil || mm.db == nil || uid == "" {
		return nil
	}

	var mp *model.MissionPoint
	res := mm.db.Where("mission_id = ? AND uid = ?", missionId, uid).Take(&mp)

	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	res = mm.db.Where("id = ?", mp.ID).Delete(&model.MissionPoint{})

	if res.Error != nil {
		mm.logger.Error("delete point error", slog.Any("error", res.Error))
		return nil
	}

	c := &model.Change{
		Type:        "REMOVE_CONTENT",
		MissionID:   missionId,
		CreatorUID:  authorUID,
		ContentUID:  uid,
		CotType:     mp.Type,
		Callsign:    mp.Callsign,
		IconsetPath: mp.IconsetPath,
		Color:       mp.Color,
		Lat:         mp.Lat,
		Lon:         mp.Lon,
	}

	_ = mm.Create(c)

	return c
}

func (mm *DatabaseManager) AddMissionContent(mission *model.Mission, hash string, authorUID string) bool {
	c := mm.FileQuery().Scope(mission.Scope).Hash(hash).One()

	if c == nil {
		return false
	}

	for _, ca := range mission.Files {
		if ca.ContentID == c.ID {
			return false
		}
	}

	ca := &model.MissionFile{ContentID: c.ID, Content: c, MissionID: mission.ID, CreatorUID: authorUID}

	mission.Files = append(mission.Files, ca)
	mm.Save(mission)

	return true
}

func (mm *DatabaseManager) DeleteMissionContent(scope string, missionId uint, hash string, authorUID string) bool {
	if mm == nil || mm.db == nil || hash == "" {
		return false
	}

	c := mm.FileQuery().Scope(scope).Hash(hash).One()

	if c == nil {
		return false
	}

	res := mm.db.Where("mission_id = ? and content_id = ?", missionId, c.ID).Delete(&model.MissionFile{})

	if res.RowsAffected > 0 {
		mm.UpdateMissionChanged(missionId)
		return true
	}

	return false
}

func (mm *DatabaseManager) GetChanges(missionId uint, after time.Time) []*model.Change {
	var m []*model.Change

	err := mm.db.Where("mission_id = ? and created_at > ?", missionId, after).Order("created_at DESC").
		Find(&m).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return m
}

func (mm *DatabaseManager) GetFiles() []*model.Content {
	var m []*model.Content

	err := mm.db.Order("created_at DESC").
		Find(&m).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return m
}

func (mm *DatabaseManager) DeleteFile(id uint) {
	mm.db.Where("id = ?", id).Delete(&model.Content{})
	mm.db.Where("content_id = ?", id).Delete(&model.MissionFile{})
}
