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
	p.callbacks.Range(func(key, value any) bool {
		if fn, ok := value.(func(msg V) bool); ok {
			go func() {
				if !fn(msg) {
					p.callbacks.Delete(key)
				}
			}()
		}

		return true
	})
}

func (p *Callback[V]) AddCallback(name string, fn func(msg V) bool) {
	p.callbacks.Store(name, fn)
}

func (p *Callback[V]) RemoveCallback(name string) bool {
	_, found := p.callbacks.LoadAndDelete(name)

	return found
}
