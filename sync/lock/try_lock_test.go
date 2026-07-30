package lock_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/lock"
)

func TestTryLock(t *testing.T) {
	l := lock.NewTryLock()
	isTrue(t, l.TryLock(t.Name()))
	isFalse(t, l.TryLock(t.Name()))
	l.Unlock(t.Name())
}

func TestTryLock_RunInLock(t *testing.T) {
	l := lock.NewTryLock()
	ch := make(chan struct{})
	key := t.Name()
	n := 3
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			<-ch
			errs[i] = l.RunInLock(key, func() error {
				time.Sleep(10 * time.Millisecond)
				return nil
			})
		})
	}

	close(ch)
	wg.Wait()

	var errCount int
	for _, err := range errs {
		if err != nil {
			errCount++
			isTrue(t, errors.Is(err, lock.ErrLocked))
		}
	}
	isTrue(t, errCount == n-1)
}
