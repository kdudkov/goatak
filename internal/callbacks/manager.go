package callbacks

import (
	"sync"
)

type Callback[V any] struct {
	callbacks sync.Map
}

func New[V any]() *Callback[V] {
	return &Callback[V]{
		callbacks: sync.Map{},
	}
}

func (p *Callback[V]) AddMessage(msg V) {
	var toRemove []any

	p.callbacks.Range(func(key, value any) bool {
		if fn, ok := value.(func(msg V) bool); ok {
			go func() {
				if !fn(msg) {
					toRemove = append(toRemove, key)
				}
			}()
		}

		return true
	})

	for _, key := range toRemove {
		p.callbacks.Delete(key)
	}
}

func (p *Callback[V]) Subscribe(name string, fn func(msg V) bool) {
	p.callbacks.Store(name, fn)
}

func (p *Callback[V]) Unsubscribe(name string) bool {
	_, found := p.callbacks.LoadAndDelete(name)

	return found
}
