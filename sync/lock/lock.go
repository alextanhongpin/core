package lock

import (
	"errors"
	"runtime"
	"sync"
	"weak"
)

var (
	ErrNotFound = errors.New("lock: not found")
	ErrExpired  = errors.New("lock: expired")
)

type unlocker interface {
	Unlock()
}

// A raw mutex cannot be weakly referenced.
// The standard pattern is to wrap the lock inside an allocated object.
// This object must be allocated on the heap to be tracked by the GC.
type entry struct {
	mu sync.Mutex
}

type KeyLock struct {
	mu    sync.Mutex
	locks map[string]weak.Pointer[entry]
}

func New() *KeyLock {
	return &KeyLock{
		locks: make(map[string]weak.Pointer[entry]),
	}
}

func (l *KeyLock) Size() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return len(l.locks)
}

func (l *KeyLock) Has(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if lock := l.locks[key].Value(); lock != nil {
		return true
	}

	delete(l.locks, key)
	return false
}

func (l *KeyLock) Lock(key string) unlocker {
	l.mu.Lock()
	defer l.mu.Unlock()

	if wp, exists := l.locks[key]; exists {
		if e := wp.Value(); e != nil {
			e.mu.Lock()
			return &e.mu
		}
	}

	e := &entry{}
	l.locks[key] = weak.Make(e)
	runtime.AddCleanup(e, func(key string) {
		l.mu.Lock()
		defer l.mu.Unlock()

		// Double check that another thread hasn't already overwritten this key.
		if wp, exists := l.locks[key]; exists && wp.Value() == nil {
			delete(l.locks, key)
		}
	}, key)

	e.mu.Lock()
	return &e.mu
}
