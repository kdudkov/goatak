package database

import (
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/model"
)

type InvitationQuery struct {
	Query[model.Invitation]
	id        uint
	missionID uint
	invitee   string
	typ       string
	full      bool
}

func NewInvitationQuery(db *gorm.DB) *InvitationQuery {
	return &InvitationQuery{
		Query: Query[model.Invitation]{
			db:     db,
			limit:  100,
			offset: 0,
			order:  "invitation.created_at",
		},
	}
}

func (q *InvitationQuery) Order(s string) *InvitationQuery {
	q.order = s
	return q
}

func (q *InvitationQuery) Limit(n int) *InvitationQuery {
	q.limit = n
	return q
}

func (q *InvitationQuery) Offset(n int) *InvitationQuery {

	q.offset = n
	return q
}

func (q *InvitationQuery) Id(id uint) *InvitationQuery {
	q.id = id
	return q
}

func (q *InvitationQuery) Mission(id uint) *InvitationQuery {
	q.missionID = id
	return q
}

func (q *InvitationQuery) Invitee(uid string) *InvitationQuery {
	q.invitee = uid
	return q
}

func (q *InvitationQuery) Type(s string) *InvitationQuery {
	q.typ = s
	return q
}

func (q *InvitationQuery) Full() *InvitationQuery {
	q.full = true
	return q
}

func (q *InvitationQuery) where() *gorm.DB {
	tx := q.db

	if q.id != 0 {
		tx = tx.Where("id = ?", q.id)
	}

	if q.missionID != 0 {
		tx = tx.Where("mission_id = ?", q.missionID)
	}

	if q.invitee != "" {
		tx = tx.Where("invitee = ?", q.invitee)
	}

	if q.typ != "" {
		tx = tx.Where("type = ?", q.typ)
	}

	if q.full {
		tx = tx.Joins("Mission")
	}

	return tx
}

func (q *InvitationQuery) Get() []*model.Invitation {
	return q.get(q.where().Model(&model.Invitation{}))
}

func (q *InvitationQuery) One() *model.Invitation {
	return q.one(q.where().Model(&model.Invitation{}))
}

func (q *InvitationQuery) Update(updates map[string]any) error {
	return q.updateOrError(q.where().Model(&model.Invitation{}), updates)
}

func (q *InvitationQuery) Delete() error {
	return q.where().Delete(&model.Invitation{}).Error
}
