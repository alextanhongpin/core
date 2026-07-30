package lock

import (
	"errors"
	"sync"
)

var ErrLocked = errors.New("already locked")

type TryLock struct {
	mu    sync.Mutex
	locks map[string]struct{}
}

func NewTryLock() *TryLock {
	return &TryLock{
		locks: make(map[string]struct{}),
	}
}

func (l *TryLock) TryLock(key string) bool {
	l.mu.Lock()
	_, ok := l.locks[key]
	if !ok {
		l.locks[key] = struct{}{}
	}
	l.mu.Unlock()

	return !ok
}

func (l *TryLock) Unlock(key string) {
	l.mu.Lock()
	delete(l.locks, key)
	l.mu.Unlock()
}

func (l *TryLock) RunInLock(key string, fn func() error) error {
	if l.TryLock(key) {
		defer l.Unlock(key)
		return fn()
	}

	return ErrLocked
}
