package main

import (
	"sync"

	"github.com/kdudkov/goatak/model"
)

type ItemsRepo struct {
	items sync.Map
}

func NewItemsRepo() *ItemsRepo {
	return new(ItemsRepo)
}

func (r *ItemsRepo) Store(i *model.Item) {
	if i != nil {
		r.items.Store(i.GetUID(), i)
	}
}
func (r *ItemsRepo) Get(uid string) *model.Item {
	if v, ok := r.items.Load(uid); ok {
		return v.(*model.Item)
	}
	return nil
}

func (r *ItemsRepo) Remove(uid string) {
	r.items.Delete(uid)
}

func (r *ItemsRepo) ForEach(f func(item *model.Item) bool) {
	r.items.Range(func(_, value any) bool {
		i := value.(*model.Item)
		return f(i)
	})
}
