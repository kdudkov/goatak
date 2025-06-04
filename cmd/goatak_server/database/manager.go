package database

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
	"github.com/kdudkov/goatak/pkg/util"
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
	return NewMissionQuery(mm.db)
}

func (mm *DatabaseManager) ResourceQuery() *ResourceQuery {
	return NewResourceQuery(mm.db)
}

func (mm *DatabaseManager) SubscriptionQuery() *SubscriptionQuery {
	return NewSubscriptionQuery(mm.db)
}

func (mm *DatabaseManager) InvitationQuery() *InvitationQuery {
	return NewInvitationQuery(mm.db)
}

func (mm *DatabaseManager) PointQuery() *PointQuery {
	return NewPointQuery(mm.db)
}

func (mm *DatabaseManager) DeviceQuery() *DeviceQuery {
	return NewDeviceQuery(mm.db)
}

func (mm *DatabaseManager) CertsQuery() *CertQuery {
	return NewCertQuery(mm.db)
}

func (mm *DatabaseManager) ProfileQuery() *ProfileQuery {
	return NewProfileQuery(mm.db)
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
		&model.Device{},
		&model.Certificate{},
		&model.Profile{},
	); err != nil {
		return err
	}

	return nil
}

func (mm *DatabaseManager) UpdateMissionChanged(id uint) {
	mm.MissionQuery().Id(id).Update(map[string]any{"updated_at": time.Now()})
}

func (mm *DatabaseManager) UpdateContentTool(id uint, tool string) {
	mm.MissionQuery().Id(id).Update(map[string]any{"tool": tool})
}

func (mm *DatabaseManager) AddMissionPoint(mission *model.Mission, msg *cot.CotMessage) *model.Change {
	if mission == nil {
		return nil
	}

	// just update point if it is already in mission
	for _, p := range mission.Points {
		if p.UID == msg.GetUID() {
			p.UpdateFromMsg(msg)
			mm.Save(p)
			return nil
		}
	}

	point := mm.PointQuery().UID(msg.GetUID()).One()

	if point == nil {
		point = &model.Point{UID: msg.GetUID()}
	}

	point.UpdateFromMsg(msg)
	mm.Save(point)

	mm.db.Model(mission).Association("Points").Append(point)

	// todo: use sender uid, not parent
	parent, _ := msg.GetParent()

	c := &model.Change{
		Type:           model.CHANGE_TYPE_ADD,
		MissionID:      mission.ID,
		CreatorUID:     parent,
		ContentUID:     msg.GetUID(),
		MissionPointID: &point.ID,
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
		Type:           model.CHANGE_TYPE_REMOVE,
		MissionID:      mission.ID,
		CreatorUID:     authorUID,
		ContentUID:     uid,
		MissionPointID: &point.ID,
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
		Type:        model.CHANGE_TYPE_ADD,
		MissionID:   mission.ID,
		CreatorUID:  authorUID,
		ContentHash: hash,
		ResourceID:  &res.ID,
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
		Type:        model.CHANGE_TYPE_REMOVE,
		MissionID:   mission.ID,
		CreatorUID:  authorUID,
		ContentHash: hash,
		ResourceID:  &res.ID,
	}

	_ = mm.Create(c)

	return c
}

func (mm *DatabaseManager) GetChanges(missionId uint, after time.Time, squashed bool) []*model.Change {
	var ch []*model.Change

	err := mm.db.Where("changes.mission_id = ? and changes.created_at > ?", missionId, after).
		Joins("MissionPoint").
		Joins("Resource").
		Order("changes.created_at DESC").
		Find(&ch).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	if !squashed {
		return ch
	}

	uids := util.NewStringSet()

	n := 0
	ch1 := make([]*model.Change, 0, len(ch))

	for _, c := range ch {
		key := util.FirstString(c.ContentUID, c.ContentHash)

		if uids.Has(key) {
			continue
		}
		uids.Add(key)

		if c.Type != model.CHANGE_TYPE_REMOVE {
			ch1 = append(ch1, c)
		}
		n++
	}

	return ch1
}
