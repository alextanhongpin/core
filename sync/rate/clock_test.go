package rate_test

import (
	"github.com/alextanhongpin/core/sync/rate"
	"sync"
	"testing"
	"time"
)

func TestConcurrentClockChanges(t *testing.T) {
	e := rate.NewErrors(time.Second)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 1000 {
			e.SetNow(time.Now)
			e.SetNow(nil)
		}
	}()
	go func() {
		defer wg.Done()
		for range 1000 {
			e.Success().Inc()
			e.Failure().Inc()
			e.Rate()
		}
	}()
	wg.Wait()
}
