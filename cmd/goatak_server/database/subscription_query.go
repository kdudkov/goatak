package database

import (
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/internal/model"
)

type SubscriptionQuery struct {
	Query[model.Subscription]
	id        uint
	missionID uint
	clientUID string
}

func NewSubscriptionQuery(db *gorm.DB) *SubscriptionQuery {
	return &SubscriptionQuery{
		Query: Query[model.Subscription]{
			db:     db,
			limit:  100,
			offset: 0,
			order:  "client_uid",
		},
	}
}

func (q *SubscriptionQuery) Order(s string) *SubscriptionQuery {
	q.order = s
	return q
}

func (q *SubscriptionQuery) Limit(n int) *SubscriptionQuery {
	q.limit = n
	return q
}

func (q *SubscriptionQuery) Offset(n int) *SubscriptionQuery {

	q.offset = n
	return q
}

func (q *SubscriptionQuery) Id(id uint) *SubscriptionQuery {
	q.id = id
	return q
}

func (q *SubscriptionQuery) Mission(id uint) *SubscriptionQuery {
	q.missionID = id
	return q
}

func (q *SubscriptionQuery) Client(uid string) *SubscriptionQuery {
	q.clientUID = uid
	return q
}

func (q *SubscriptionQuery) where() *gorm.DB {
	tx := q.db

	if q.id != 0 {
		tx = tx.Where("id = ?", q.id)
	}

	if q.missionID != 0 {
		tx = tx.Where("mission_id = ?", q.missionID)
	}

	if q.clientUID != "" {
		tx = tx.Where("client_uid = ?", q.clientUID)
	}

	return tx
}

func (q *SubscriptionQuery) Get() []*model.Subscription {
	return q.get(q.where().Model(&model.Subscription{}))
}

func (q *SubscriptionQuery) One() *model.Subscription {
	return q.one(q.where().Model(&model.Subscription{}))
}

func (q *SubscriptionQuery) Update(updates map[string]any) error {
	return q.updateOrError(q.where().Model(&model.Subscription{}), updates)
}

func (q *SubscriptionQuery) Delete() error {
	return q.where().Delete(&model.Subscription{}).Error
}
