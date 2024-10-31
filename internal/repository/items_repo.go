package repository

import (
	"sync"
	"time"

	"github.com/kdudkov/goutils/callback"

	"github.com/kdudkov/goatak/pkg/model"
)

type ItemsMemoryRepo struct {
	items                         sync.Map
	lastSeenContactOfflineTimeout time.Duration
	changeCb                      *callback.Callback[*model.Item]
	deleteCb                      *callback.Callback[string]
}

func NewItemsMemoryRepo(tm ...time.Duration) *ItemsMemoryRepo {
	defaultTm := time.Minute * 5

	if len(tm) > 0 && tm[0] > 0 {
		defaultTm = tm[0]
	}

	return &ItemsMemoryRepo{
		items:                         sync.Map{},
		lastSeenContactOfflineTimeout: defaultTm,
		changeCb:                      callback.New[*model.Item](),
		deleteCb:                      callback.New[string](),
	}
}

func (r *ItemsMemoryRepo) Start() error {
	go r.cleaner()

	return nil
}

func (r *ItemsMemoryRepo) Stop() {
	// no-op
}

func (r *ItemsMemoryRepo) ChangeCallback() *callback.Callback[*model.Item] {
	return r.changeCb
}

func (r *ItemsMemoryRepo) DeleteCallback() *callback.Callback[string] {
	return r.deleteCb
}

func (r *ItemsMemoryRepo) Store(i *model.Item) {
	if i != nil {
		r.items.Store(i.GetUID(), i)
		r.changeCb.AddMessage(i)
	}
}

func (r *ItemsMemoryRepo) Get(uid string) *model.Item {
	if v, ok := r.items.Load(uid); ok {
		return v.(*model.Item)
	}

	return nil
}

func (r *ItemsMemoryRepo) GetByCallsign(callsign string) *model.Item {
	var i *model.Item

	r.ForEach(func(item *model.Item) bool {
		if item.GetCallsign() == callsign {
			i = item

			return false
		}

		return true
	})

	return i
}

func (r *ItemsMemoryRepo) Remove(uid string) {
	if _, ok := r.items.LoadAndDelete(uid); ok {
		r.deleteCb.AddMessage(uid)
	}
}

func (r *ItemsMemoryRepo) ForEach(f func(item *model.Item) bool) {
	r.items.Range(func(_, value any) bool {
		i := value.(*model.Item)

		return f(i)
	})
}

func (r *ItemsMemoryRepo) GetCallsign(uid string) string {
	i := r.Get(uid)
	if i != nil {
		return i.GetCallsign()
	}

	return ""
}

func (r *ItemsMemoryRepo) cleaner() {
	for range time.Tick(time.Second) {
		r.cleanOldUnits()
	}
}

func (r *ItemsMemoryRepo) cleanOldUnits() {
	toDelete := make([]string, 0)

	r.ForEach(func(item *model.Item) bool {
		switch item.GetClass() {
		case model.UNIT, model.POINT:
			if item.IsOld() {
				toDelete = append(toDelete, item.GetUID())
			}
		case model.CONTACT:
			if item.IsOld() {
				toDelete = append(toDelete, item.GetUID())
			} else if item.IsOnline() && time.Since(item.GetLastSeen()) > r.lastSeenContactOfflineTimeout {
				item.SetOffline()
				r.changeCb.AddMessage(item)
			}
		}

		return true
	})

	for _, uid := range toDelete {
		r.Remove(uid)
	}
}
