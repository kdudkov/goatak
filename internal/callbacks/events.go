package callbacks

import (
	"sync"

	"github.com/google/uuid"
)

type Events struct {
	cb sync.Map
}

type EventHandler struct {
	cb sync.Map
}

func NewEvents() *Events {
	return &Events{cb: sync.Map{}}
}

func (e *Events) On(key string, f func(data any) bool) {
	if k, ok := e.cb.Load(key); !ok {
		eh := &EventHandler{cb: sync.Map{}}
		k1, _ := e.cb.LoadOrStore(key, eh)

		if eh1, ok := k1.(*EventHandler); ok {
			eh1.add(f)
		}
	} else {
		if eh, ok := k.(*EventHandler); ok {
			eh.add(f)
		}
	}
}

func (e *Events) Add(key string, data any) {
	if k, ok := e.cb.Load(key); ok {
		if eh, ok := k.(*EventHandler); ok {
			eh.fire(data)
		}
	}
}

func (eh *EventHandler) fire(data any) {
	eh.cb.Range(func(key, value any) bool {
		if fn, ok := value.(func(data any) bool); ok {
			go func() {
				if !fn(data) {
					eh.remove(key.(string))
				}
			}()
		}

		return true
	})
}

func (eh *EventHandler) add(fn func(data any) bool) {
	eh.cb.Store(uuid.NewString(), fn)
}

func (eh *EventHandler) remove(name string) bool {
	_, found := eh.cb.LoadAndDelete(name)

	return found
}
