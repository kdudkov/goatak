package callbacks

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/exp/rand"
)

func TestRemove(t *testing.T) {
	cb := New[string]()

	for i := 0; i < 30; i++ {
		cb.Subscribe(fmt.Sprintf("cb_%d", i), func(msg string) bool {
			if rand.Intn(1000) == 1 {
				fmt.Printf("remove\n")
				return false
			}

			return true
		})
	}

	n := 10

	ctx, cancel := context.WithCancel(context.Background())

	wg := new(sync.WaitGroup)

	for i := 0; i < n; i++ {
		wg.Add(1)

		go func() {
			for ctx.Err() == nil {
				cb.AddMessage("aaa")

				time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
			}

			wg.Done()
		}()
	}

	time.Sleep(time.Second * 5)
	cancel()

	wg.Wait()

}
