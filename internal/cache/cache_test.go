package cache

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func BenchmarkCache(b *testing.B) {
	c := NewWithTTL[*time.Time](time.Millisecond*100, func(key string) *time.Time {
		t := time.Now()
		return &t
	})

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < b.N; i++ {
		_ = c.Load(strconv.Itoa(r.Intn(50)))
	}
}

func TestCache(t *testing.T) {
	ttl := time.Millisecond * 10
	c := NewWithTTL[*time.Time](ttl, func(key string) *time.Time {
		t := time.Now()
		return &t
	})

	wg := new(sync.WaitGroup)

	go func() {
		c.Clean()
	}()

	for n := 0; n < 50; n++ {
		wg.Add(1)
		go func() {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))

			for i := 0; i < 100000; i++ {
				res := c.Load(strconv.Itoa(r.Intn(1000)))

				assert.NotNil(t, res)
				assert.Less(t, time.Since(*res), ttl*time.Duration(2))
			}
			wg.Done()
		}()
	}

	wg.Wait()
}
