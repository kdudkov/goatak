package database

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/kdudkov/goatak/internal/model"
)

func (mm *DatabaseManager) Subscribe(user *model.User, mission *model.Mission, uid, password string) (*model.Subscription, error) {
	if mission.InviteOnly {
		return nil, fmt.Errorf("Illegal attempt to subscribe to invite only mission!")
	}

	if mission.Password != "" && password != mission.Password {
		return nil, fmt.Errorf("Illegal attempt to subscribe to mission! Password did not match.")
	}

	return mm.subscribe(mission.ID, uid, user.GetLogin(), false)
}

func (mm *DatabaseManager) subscribe(missionID uint, clientUID string, username string, creator bool) (*model.Subscription, error) {
	var s *model.Subscription

	role := "MISSION_SUBSCRIBER"
	if creator {
		role = "MISSION_CREATOR"
	}

	err := mm.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("mission_id = ? AND client_uid = ?", missionID, clientUID).Find(&s).Error

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		s.MissionID = missionID
		s.ClientUID = clientUID
		s.Username = username
		s.Role = role

		return tx.Save(s).Error
	})

	return s, err
}

func (mm *DatabaseManager) GetSubscriptions(missionId uint) []*model.Subscription {
	var s []*model.Subscription

	result := mm.db.Where("mission_id = ?", missionId).Find(&s)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	return s
}

func (mm *DatabaseManager) GetSubscribers(missionId uint) []string {
	subscriptions := mm.GetSubscriptions(missionId)

	res := make([]string, len(subscriptions))

	for i, s := range subscriptions {
		res[i] = s.ClientUID
	}

	return res
}

func (mm *DatabaseManager) GetSubscription(missionId uint, uid string) *model.Subscription {
	var s *model.Subscription

	result := mm.db.Where("mission_id = ? AND client_uid = ?", missionId, uid).Take(&s)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	return s
}

func (mm *DatabaseManager) DeleteSubscription(missionID uint, uid string) {
	mm.db.Where("mission_id = ? AND client_uid = ?", missionID, uid).Delete(&model.Subscription{})
}

func (mm *DatabaseManager) Invite(s *model.Invitation) (*model.Invitation, error) {
	var old *model.Invitation

	result := mm.db.Where("mission_id = ? AND invitee = ? AND typ = ?", s.MissionID, s.Invitee, s.Typ).Take(&old)

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	old.MissionID = s.MissionID
	old.Typ = s.Typ
	old.Invitee = s.Invitee
	old.Role = s.Role
	old.CreatorUID = s.CreatorUID

	return old, mm.db.Save(old).Error
}

func (mm *DatabaseManager) DeleteInvitation(missionId uint, uid string, typ string) {
	mm.db.Where("mission_id = ? AND invitee = ? AND typ = ?", missionId, uid, typ).Delete(&model.Invitation{})
}

func (mm *DatabaseManager) GetInvitations(uid string) []string {
	var m []*model.Invitation

	result := mm.db.Where("invitee = ?", uid).Joins("Mission").Find(&m)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	res := make([]string, len(m))

	for i, s := range m {
		res[i] = s.Mission.Name
	}

	return res
}
