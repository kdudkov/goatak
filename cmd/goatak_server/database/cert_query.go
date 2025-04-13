package database

import (
	"gorm.io/gorm"

	"github.com/kdudkov/goatak/pkg/model"
)

type CertQuery struct {
	Query[model.Certificate]
	uid   string
	login string
	sn    string
}

func NewCertQuery(db *gorm.DB) *CertQuery {
	return &CertQuery{
		Query: Query[model.Certificate]{
			db:     db,
			limit:  100,
			offset: 0,
			order:  "login DESC",
		},
	}
}

func (q *CertQuery) Order(s string) *CertQuery {
	q.order = s
	return q
}

func (q *CertQuery) Limit(n int) *CertQuery {
	q.limit = n
	return q
}

func (q *CertQuery) Offset(n int) *CertQuery {

	q.offset = n
	return q
}

func (q *CertQuery) UID(uid string) *CertQuery {
	q.uid = uid
	return q
}

func (q *CertQuery) Login(login string) *CertQuery {
	q.login = login
	return q
}

func (q *CertQuery) SN(sn string) *CertQuery {
	q.sn = sn
	return q
}

func (q *CertQuery) where() *gorm.DB {
	tx := q.db

	if q.uid != "" {
		tx = tx.Where("uid = ?", q.uid)
	}

	if q.login != "" {
		tx = tx.Where("login = ?", q.login)
	}

	if q.sn != "" {
		tx = tx.Where("serial = ?", q.sn)
	}

	return tx
}

func (q *CertQuery) Get() []*model.Certificate {
	return q.get(q.where().Model(&model.Certificate{}))
}

func (q *CertQuery) One() *model.Certificate {
	return q.one(q.where().Model(&model.Certificate{}))
}

func (q *CertQuery) Count() int64 {
	return q.count(q.where().Model(&model.Certificate{}))
}

func (q *CertQuery) Update(updates map[string]any) error {
	return q.updateOrError(q.where().Model(&model.Certificate{}), updates)
}
