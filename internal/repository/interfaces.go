package repository

import (
	internal "github.com/kdudkov/goatak/internal/model"
	"github.com/kdudkov/goatak/pkg/model"
)

type UserRepository interface {
	Start() error
	Stop()
	CheckUserAuth(user, password string) bool
	UserIsValid(user, sn string) bool
	GetUser(username string) *internal.User
}

type ItemsRepository interface {
	Start() error
	Stop()
	Store(i *model.Item)
	Get(uid string) *model.Item
	GetByCallsign(callsign string) *model.Item
	Remove(uid string)
	ForEach(f func(item *model.Item) bool)
	GetCallsign(uid string) string
	ForMission(name string) []*model.Item
}

type FeedsRepository interface {
	Start() error
	Stop()
	Store(i *model.Feed2)
	Get(uid string) *model.Feed2
	Remove(uid string)
	ForEach(f func(item *model.Feed2) bool)
}
