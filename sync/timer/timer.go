package timer

import (
	"sync"
	"time"
)

func SetInterval(fn func(), duration time.Duration) func() {
	var wg sync.WaitGroup
	wg.Add(1)

	done := make(chan struct{})
	go func() {

		defer wg.Done()
		t := time.NewTicker(duration)
		defer t.Stop()

		for {
			select {
			case <-done:
				return
			case <-t.C:
				fn()
			}
		}
	}()

	return sync.OnceFunc(func() {
		close(done)
		wg.Wait()
	})
}

func SetTimeout(fn func(), duration time.Duration) func() {
	stop := time.AfterFunc(duration, fn).Stop
	return sync.OnceFunc(func() {
		_ = stop()
	})
}
