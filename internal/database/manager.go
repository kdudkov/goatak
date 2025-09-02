package database

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
	"github.com/kdudkov/goatak/pkg/util"
)

type DatabaseManager struct {
	db     *gorm.DB
	logger *slog.Logger
}

func New(db *gorm.DB) *DatabaseManager {
	m := &DatabaseManager{
		db:     db,
		logger: slog.With("logger", "dbm"),
	}

	return m
}

func (mm *DatabaseManager) AddDefaults() {
	if mm.ProfileQuery().Count() == 0 {
		defaultPrefs := map[string]string{
			"deviceProfileEnableOnConnect":  "true",
			"speed_unit_pref":               "1",
			"alt_unit_pref":                 "1",
			"saHasPhoneNumber":              "false",
			"alt_display_pref":              "MSL",
			"coord_display_pref":            "DD",
			"rab_north_ref_pref":            "1",
			"rab_brg_units":                 "0",
			"rab_nrg_units":                 "1",
			"displayServerConnectionWidget": "true",
			"frame_limit":                   "1",
			"hidePreferenceItem_deviceProfileEnableOnConnect": "true",
		}

		if err := mm.Save(&model.Profile{Login: "*", UID: "*", Options: defaultPrefs}); err != nil {
			mm.logger.Error("error create profile", slog.Any("error", err))
		}
	}
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

func (mm *DatabaseManager) ForceSave(s any) error {
	if mm == nil || mm.db == nil {
		return nil
	}

	err := mm.db.Clauses(clause.OnConflict{UpdateAll: true}).Save(s).Error

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

func (mm *DatabaseManager) ChangeQuery() *ChangeQuery {
	return NewChangeQuery(mm.db)
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

func (mm *DatabaseManager) FeedQuery() *FeedQuery {
	return NewFeedQuery(mm.db)
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
		&model.Feed2{},
	); err != nil {
		return err
	}

	return nil
}

func (mm *DatabaseManager) UpdateMissionChanged(id uint) error {
	return mm.MissionQuery().Id(id).Update(map[string]any{"updated_at": time.Now()})
}

func (mm *DatabaseManager) UpdateContentTool(id uint, tool string) error {
	return mm.MissionQuery().Id(id).Update(map[string]any{"tool": tool})
}

func (mm *DatabaseManager) AddMissionPoint(mission *model.Mission, msg *cot.CotMessage) (*model.Change, error) {
	if mission == nil {
		return nil, nil
	}

	// just update point if it is already in mission
	for _, p := range mission.Points {
		if p.UID == msg.GetUID() {
			p.UpdateFromMsg(msg)

			return nil, mm.Save(p)
		}
	}

	point := mm.PointQuery().UID(msg.GetUID()).One()

	if point == nil {
		point = &model.Point{UID: msg.GetUID()}
	}

	point.UpdateFromMsg(msg)
	if err := mm.Save(point); err != nil {
		return nil, err
	}

	if err := mm.db.Model(mission).Association("Points").Append(point); err != nil {
		return nil, err
	}

	// todo: use sender uid, not parent
	parent, _ := msg.GetParent()

	c := &model.Change{
		Type:           model.CHANGE_TYPE_ADD,
		MissionID:      mission.ID,
		CreatorUID:     parent,
		ContentUID:     msg.GetUID(),
		MissionPointID: &point.ID,
	}

	err := mm.Create(c)

	return c, err
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
	ch := mm.ChangeQuery().Mission(missionId).After(after).Get()

	if !squashed {
		return ch
	}

	uids := util.NewStringSet()

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
	}

	return ch1
}
