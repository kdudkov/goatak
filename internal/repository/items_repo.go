package repository

import (
	"sync"

	"github.com/kdudkov/goatak/pkg/model"
)

type ItemsMemoryRepo struct {
	items sync.Map
}

func NewItemsMemoryRepo() *ItemsMemoryRepo {
	return new(ItemsMemoryRepo)
}

func (r *ItemsMemoryRepo) Start() error {
	return nil
}

func (r *ItemsMemoryRepo) Stop() {
	// no-op
}

func (r *ItemsMemoryRepo) Store(i *model.Item) {
	if i != nil {
		r.items.Store(i.GetUID(), i)
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
	r.items.Delete(uid)
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

func (r *ItemsMemoryRepo) ForMission(name string) []*model.Item {
	var res []*model.Item

	r.items.Range(func(_, value any) bool {
		i := value.(*model.Item)

		if i.HasMission(name) {
			res = append(res, i)
		}

		return true
	})

	return res
}
