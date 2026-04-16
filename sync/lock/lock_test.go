package lock_test

import (
	"runtime"
	"testing"

	"github.com/alextanhongpin/core/sync/lock"
)

func TestLock(t *testing.T) {
	t.Run("weak pointer cleared", func(t *testing.T) {
		key := t.Name()

		l := lock.New()
		lk := l.Lock(key)
		lk.Unlock()

		isEqual(t, 1, l.Size())
		isTrue(t, l.Has(key))

		runtime.GC()
		runtime.GC()

		// Lock has cleared by garbage collection.
		isFalse(t, l.Has(key))
	})

	t.Run("strong pointer remains", func(t *testing.T) {
		key := t.Name()

		l := lock.New()
		lk := l.Lock(key)
		defer lk.Unlock()

		isEqual(t, 1, l.Size())
		isTrue(t, l.Has(key))

		runtime.GC()
		runtime.GC()

		// Lock remains because there is a strong pointer.
		isTrue(t, l.Has(key))
	})
}

func isEqual(t *testing.T, want, got any) {
	t.Helper()
	if want != got {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func isTrue(t *testing.T, got bool) {
	t.Helper()

	isEqual(t, true, got)
}

func isFalse(t *testing.T, got bool) {
	t.Helper()

	isEqual(t, false, got)
}

func noError(t *testing.T, err error) {
	t.Helper()

	isEqual(t, nil, err)
}
