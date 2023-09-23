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

func (r *ItemsMemoryRepo) Remove(uid string) {
	r.items.Delete(uid)
}

func (r *ItemsMemoryRepo) ForEach(f func(item *model.Item) bool) {
	r.items.Range(func(_, value any) bool {
		i := value.(*model.Item)
		return f(i)
	})
}
