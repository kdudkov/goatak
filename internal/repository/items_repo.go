package repository

import (
	"sync"

	"github.com/kdudkov/goatak/pkg/model"
)

type ItemsFileRepo struct {
	items sync.Map
}

func NewItemsFileRepo() *ItemsFileRepo {
	return new(ItemsFileRepo)
}

func (r *ItemsFileRepo) Start() error {
	return nil
}

func (r *ItemsFileRepo) Stop() {
	// no-op
}

func (r *ItemsFileRepo) Store(i *model.Item) {
	if i != nil {
		r.items.Store(i.GetUID(), i)
	}
}
func (r *ItemsFileRepo) Get(uid string) *model.Item {
	if v, ok := r.items.Load(uid); ok {
		return v.(*model.Item)
	}
	return nil
}

func (r *ItemsFileRepo) Remove(uid string) {
	r.items.Delete(uid)
}

func (r *ItemsFileRepo) ForEach(f func(item *model.Item) bool) {
	r.items.Range(func(_, value any) bool {
		i := value.(*model.Item)
		return f(i)
	})
}
