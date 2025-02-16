package database

import (
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
	if clientUID == "" {
		return nil, nil
	}

	var s *model.Subscription

	role := "MISSION_SUBSCRIBER"
	if creator {
		role = "MISSION_CREATOR"
	}

	err := mm.db.Transaction(func(tx *gorm.DB) error {
		if ss := NewSubscriptionQuery(tx).Mission(missionID).Client(clientUID).One(); ss != nil {
			s = ss
		}

		s.MissionID = missionID
		s.ClientUID = clientUID
		s.Username = username
		s.Role = role

		return tx.Save(s).Error
	})

	return s, err
}

func (mm *DatabaseManager) GetSubscribers(missionId uint) []string {
	subscriptions := mm.SubscriptionQuery().Mission(missionId).Get()

	res := make([]string, len(subscriptions))

	for i, s := range subscriptions {
		res[i] = s.ClientUID
	}

	return res
}

func (mm *DatabaseManager) Invite(s *model.Invitation) (*model.Invitation, error) {
	var old *model.Invitation

	if i := mm.InvitationQuery().Mission(s.MissionID).Invitee(s.Invitee).Type(s.Typ).One(); i != nil {
		old = i
	}

	old.MissionID = s.MissionID
	old.Typ = s.Typ
	old.Invitee = s.Invitee
	old.Role = s.Role
	old.CreatorUID = s.CreatorUID

	return old, mm.db.Save(old).Error
}

func (mm *DatabaseManager) GetInvitations(uid string) []string {
	m := mm.InvitationQuery().Invitee(uid).Full().Get()
	res := make([]string, len(m))

	for i, s := range mm.InvitationQuery().Invitee(uid).Full().Get() {
		res[i] = s.Mission.Name
	}

	return res
}
