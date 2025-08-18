package util

import "sync"

type Named interface {
	Name() string
}

func NewHolder[T Named]() *Holder[T] {
	return &Holder[T]{
		data: sync.Map{},
	}
}

type Holder[T Named] struct {
	data sync.Map
}

func (h *Holder[T]) Get(uid string) (*T, bool) {
	if v, ok := h.data.Load(uid); ok {
		if n, ok1 := v.(*T); ok1 {
			return n, true
		}
	}

	return nil, false
}

func (h *Holder[T]) Add(c *T) {
	if c == nil {
		return
	}

	h.data.Store((*c).Name(), c)
}

func (h *Holder[T]) Remove(name string) {
	h.data.Delete(name)
}

func (h *Holder[T]) RemoveExec(name string, f func(c *T)) {
	if v, ok := h.data.LoadAndDelete(name); ok {
		if c, ok1 := v.(*T); ok1 {
			f(c)
		}
	}
}

func (h *Holder[T]) All(f func(c *T) bool) {
	h.data.Range(func(_, value any) bool {
		if c, ok := value.(*T); ok {
			return f(c)
		}

		return true
	})
}
