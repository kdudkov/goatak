package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/tools"
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

func (mm *DatabaseManager) ResourceQuery() *ResourceQuery {
	if mm == nil || mm.db == nil {
		return nil
	}

	return NewResourceQuery(mm.db)
}

func (mm *DatabaseManager) Migrate() error {
	if mm == nil || mm.db == nil {
		return fmt.Errorf("no database")
	}

	// Migrate the schema
	if err := mm.db.AutoMigrate(
		&model.Mission{},
		&model.Change{},
		&model.Point{},
		&model.Subscription{},
		&model.Invitation{},
		&model.Resource{},
	); err != nil {
		return err
	}

	return nil
}

func (mm *DatabaseManager) GetPoint(uid string) *model.Point {
	if mm == nil || mm.db == nil {
		return nil
	}

	var d *model.Point

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

	var point *model.Point

	for _, p := range mission.Points {
		if p.UID == msg.GetUID() {
			point = p
			break
		}
	}

	if point != nil {
		point.UpdateFromMsg(msg)
		mm.Save(point)
		return nil
	}

	point = &model.Point{UID: msg.GetUID()}
	point.UpdateFromMsg(msg)
	mm.Save(point)

	mm.db.Model(mission).Association("Points").Append(point)

	// todo: use sender uid, not parent
	parent, _ := msg.GetParent()

	c := &model.Change{
		Type:           "ADD_CONTENT",
		MissionID:      mission.ID,
		CreatorUID:     parent,
		ContentUID:     msg.GetUID(),
		MissionPointID: sql.NullInt32{int32(point.ID), true},
	}

	_ = mm.Create(c)

	return c
}

func (mm *DatabaseManager) DeleteMissionPoint(mission *model.Mission, uid string, authorUID string) *model.Change {
	if mm == nil || mm.db == nil || uid == "" {
		return nil
	}

	var point *model.Point

	for _, p := range mission.Points {
		if p.UID == uid {
			point = p
			break
		}
	}

	if point == nil {
		return nil
	}

	mm.db.Model(mission).Association("Points").Delete(point)

	c := &model.Change{
		Type:           "REMOVE_CONTENT",
		MissionID:      mission.ID,
		CreatorUID:     authorUID,
		ContentUID:     uid,
		MissionPointID: sql.NullInt32{int32(point.ID), true},
	}

	_ = mm.Create(c)

	return c
}

func (mm *DatabaseManager) AddMissionResource(mission *model.Mission, hash string, authorUID string) *model.Change {
	if mm == nil || mm.db == nil || hash == "" {
		return nil
	}

	var res *model.Resource

	for _, r := range mission.Resources {
		if r.Hash == hash {
			res = r
			break
		}
	}

	if res != nil {
		return nil
	}

	res = mm.ResourceQuery().Scope(mission.Scope).Hash(hash).One()

	if res == nil {
		return nil
	}

	mm.db.Model(mission).Association("Resources").Append(res)

	c := &model.Change{
		Type:        "ADD_CONTENT",
		MissionID:   mission.ID,
		CreatorUID:  authorUID,
		ContentHash: hash,
		ResourceID:  sql.NullInt32{int32(res.ID), true},
	}

	_ = mm.Create(c)

	return c
}

func (mm *DatabaseManager) DeleteMissionContent(mission *model.Mission, hash string, authorUID string) *model.Change {
	if mm == nil || mm.db == nil || hash == "" {
		return nil
	}

	var res *model.Resource

	for _, r := range mission.Resources {
		if r.Hash == hash {
			res = r
			break
		}
	}

	if res == nil {
		return nil
	}

	mm.db.Model(mission).Association("Resources").Delete(res)

	c := &model.Change{
		Type:        "REMOVE_CONTENT",
		MissionID:   mission.ID,
		CreatorUID:  authorUID,
		ContentHash: hash,
		ResourceID:  sql.NullInt32{int32(res.ID), true},
	}

	_ = mm.Create(c)

	return c
}

func (mm *DatabaseManager) GetChanges(missionId uint, after time.Time, squashed bool) []*model.Change {
	var ch []*model.Change

	err := mm.db.Where("mission_id = ? and created_at > ?", missionId, after).Order("created_at DESC").
		Find(&ch).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	if !squashed {
		return ch
	}

	uids := tools.NewStringSet()

	n := 0
	ch1 := make([]*model.Change, 0)

	for _, c := range ch {
		key := c.ContentUID
		if key == "" {
			key = c.ContentHash
		}

		if uids.Has(key) {
			continue
		}
		uids.Add(key)

		if c.Type != "REMOVE_CONTENT" {
			ch1 = append(ch1, c)
		}
		n++
	}

	return ch1
}

func (mm *DatabaseManager) GetFiles() []*model.Resource {
	var m []*model.Resource

	err := mm.db.Order("created_at DESC").
		Find(&m).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return m
}

func (mm *DatabaseManager) DeleteFile(id uint) {
	mm.db.Where("id = ?", id).Delete(&model.Resource{})
}
