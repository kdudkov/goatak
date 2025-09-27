package chat

import (
	"slices"
	"sync"
	"time"

	"github.com/kdudkov/goatak/pkg/cot"
	"github.com/kdudkov/goatak/pkg/model"
)

type Storage struct {
	msg sync.Map
}

func NewStorage() *Storage {
	return &Storage{
		msg: sync.Map{},
	}
}

func (s *Storage) Add(c *ChatMessage) {
	s.msg.Store(c.msg.GetUID(), c)
}

func (s *Storage) Start() error {
	go func() {
		for range time.Tick(time.Second * 60) {
			s.clean()
		}
	}()

	return nil
}

func (s *Storage) clean() {
	s.ForEach(func(c *ChatMessage) bool {
		if c.msg.GetStaleTime().Before(time.Now()) {
			s.msg.Delete(c.msg.GetUID())
		}

		return true
	})
}

func (s *Storage) GetFor(item *model.Item, t time.Time) []*cot.CotMessage {
	res := make([]*cot.CotMessage, 0)

	t1 := t
	if t1.IsZero() {
		t1 = time.Now().Add(- time.Hour * 24)
	}
	
	s.ForEach(func(c *ChatMessage) bool {
		if c.received.Before(t1) || c.msg.GetStaleTime().Before(time.Now()) {
			return true
		}

		if dest := c.msg.GetDetail().GetDestMission(); len(dest) > 0 {
			return true
		}

		if dest := c.msg.GetDetail().GetDestCallsign(); len(dest) > 0 {
			if slices.Contains(dest, item.GetCallsign()) {
				res = append(res, c.msg)

				return true
			}

			return true
		}

		
		if dest := c.msg.GetDetail().GetDestUid(); len(dest) > 0 {
			if slices.Contains(dest, item.GetUID()) {
				res = append(res, c.msg)

				return true
			}

			return true
		}
		
		res = append(res, c.msg)

		return true
	})

	return res
}

func (s *Storage) ForEach(f func(c *ChatMessage) bool) {
	s.msg.Range(func(_, v any) bool {
		if c, ok := v.(*ChatMessage); ok {
			return f(c)
		}

		return true
	})
}
