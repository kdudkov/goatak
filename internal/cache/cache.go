package cache

import (
	"sync"
	"time"
)

type Cache[T any] struct {
	m      sync.Map
	ttl    time.Duration
	loader func(key string) T
}

type entry[T any] struct {
	mx    sync.Mutex
	value T
	ts    time.Time
}

func NewWithTTL[T any](ttl time.Duration, loader func(key string) T) *Cache[T] {
	return &Cache[T]{
		m:      sync.Map{},
		ttl:    ttl,
		loader: loader,
	}
}

func (c *Cache[T]) Clean() {
	c.m.Range(func(key, value any) bool {
		e := value.(*entry[T])

		if !e.mx.TryLock() {
			return true
		}

		defer e.mx.Unlock()

		if time.Since(e.ts) > c.ttl*10 {
			c.m.Delete(key)
		}

		return true
	})
}

func (c *Cache[T]) Load(key string) T {
	var e *entry[T]

	if v, ok := c.m.Load(key); ok {
		e = v.(*entry[T])
	} else {
		v1, _ := c.m.LoadOrStore(key, new(entry[T]))
		e = v1.(*entry[T])
	}

	e.mx.Lock()
	defer e.mx.Unlock()

	if e.ts.IsZero() || time.Since(e.ts) > c.ttl {
		e.value = c.loader(key)
		e.ts = time.Now()
	}

	return e.value
}
