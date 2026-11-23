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

func TestTryAcquire(t *testing.T) {
	var l lock.TryLock // zero value
	key := t.Name()

	unlock, ok := l.TryAcquire(key)
	isTrue(t, ok)
	isTrue(t, unlock != nil)
	other, ok := l.TryAcquire(key)
	isFalse(t, ok)
	isTrue(t, other == nil)

	unlock()
	unlock() // idempotent
	again, ok := l.TryAcquire(key)
	isTrue(t, ok)

	// A stale unlock must not release the new holder's lock.
	unlock()
	isFalse(t, l.TryLock(key))

	// Same after a plain Unlock followed by a new acquisition.
	l.Unlock(key)
	third, ok := l.TryAcquire(key)
	isTrue(t, ok)
	again()
	isFalse(t, l.TryLock(key))
	third()
	isTrue(t, l.TryLock(key))
}

func TestTryLock_RunInLockReleasesOnlyOwn(t *testing.T) {
	l := lock.NewTryLock()
	key := t.Name()
	err := l.RunInLock(key, func() error {
		l.Unlock(key)
		isTrue(t, l.TryLock(key)) // someone else takes it
		return nil
	})
	isTrue(t, err == nil)
	isFalse(t, l.TryLock(key))
}
