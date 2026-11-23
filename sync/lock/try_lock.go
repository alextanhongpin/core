package lock

import (
	"errors"
	"sync"
)

var ErrLocked = errors.New("already locked")

// TryLock is a nonblocking keyed lock. Its zero value is ready to use and it is
// safe for concurrent use. It coordinates callers within one process only.
type TryLock struct {
	mu    sync.Mutex
	next  uint64            // last generation handed out
	locks map[string]uint64 // key -> generation of the current acquisition
}

func NewTryLock() *TryLock {
	return &TryLock{
		locks: make(map[string]uint64),
	}
}

// TryLock acquires key if it is free and reports whether it did. Release it
// with Unlock, which releases whoever holds the key; prefer TryAcquire when the
// release must only affect this acquisition.
func (l *TryLock) TryLock(key string) bool {
	_, ok := l.acquire(key)
	return ok
}

// TryAcquire acquires key if it is free. On success it returns an unlock func
// bound to this acquisition: it releases the key only if this acquisition still
// holds it, so calling it after the key was released (for example by Unlock) and
// acquired by someone else does not release their lock. The func is idempotent
// and safe for concurrent use. On failure it returns nil and false.
func (l *TryLock) TryAcquire(key string) (unlock func(), ok bool) {
	gen, ok := l.acquire(key)
	if !ok {
		return nil, false
	}
	return func() {
		l.mu.Lock()
		if l.locks[key] == gen {
			delete(l.locks, key)
		}
		l.mu.Unlock()
	}, true
}

func (l *TryLock) acquire(key string) (gen uint64, ok bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.locks == nil {
		l.locks = make(map[string]uint64)
	}
	if _, held := l.locks[key]; held {
		return 0, false
	}
	l.next++
	l.locks[key] = l.next
	return l.next, true
}

// Unlock releases key regardless of which acquisition holds it.
func (l *TryLock) Unlock(key string) {
	l.mu.Lock()
	delete(l.locks, key)
	l.mu.Unlock()
}

// RunInLock runs fn while holding key and releases only its own acquisition
// afterwards. It returns ErrLocked without calling fn if key is held.
func (l *TryLock) RunInLock(key string, fn func() error) error {
	if unlock, ok := l.TryAcquire(key); ok {
		defer unlock()
		return fn()
	}

	return ErrLocked
}
