package callbacks

import (
	"sync"

	"github.com/google/uuid"
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
					p.Unsubscribe(key.(string))
				}
			}()
		}

		return true
	})
}

func (p *Callback[V]) Add(fn func(msg V) bool) {
	p.callbacks.Store(uuid.NewString(), fn)
}

func (p *Callback[V]) Subscribe(name string, fn func(msg V) bool) {
	p.callbacks.Store(name, fn)
}

func (p *Callback[V]) Unsubscribe(name string) bool {
	_, found := p.callbacks.LoadAndDelete(name)

	return found
}
