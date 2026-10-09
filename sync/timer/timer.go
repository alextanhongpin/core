package timer

import (
	"sync"
	"time"
)

// SetInterval runs fn serially on each tick. The returned stop function is
// concurrent-safe and waits for an active callback, so fn must not call it.
// A nonpositive duration panics in the caller.
func SetInterval(fn func(), duration time.Duration) func() {
	t := time.NewTicker(duration)
	var wg sync.WaitGroup
	wg.Add(1)

	done := make(chan struct{})
	go func() {

		defer wg.Done()
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

// SetTimeout schedules fn once. Cancellation is concurrent-safe but does not
// wait for a callback that has already started.
func SetTimeout(fn func(), duration time.Duration) func() {
	stop := time.AfterFunc(duration, fn).Stop
	return sync.OnceFunc(func() {
		_ = stop()
	})
}
