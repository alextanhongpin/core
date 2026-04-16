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

type KeyLock struct {
	mu   sync.Mutex
	data map[string]weak.Pointer[sync.Mutex]
}

func New() *KeyLock {
	return &KeyLock{
		data: make(map[string]weak.Pointer[sync.Mutex]),
	}
}

func (l *KeyLock) Size() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return len(l.data)
}

func (l *KeyLock) Has(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if lock := l.data[key].Value(); lock != nil {
		return true
	}

	delete(l.data, key)
	return false
}

func (l *KeyLock) Lock(key string) unlocker {
	l.mu.Lock()
	defer l.mu.Unlock()

	if lock := l.data[key].Value(); lock != nil {
		lock.Lock()
		return lock
	}

	lock := new(sync.Mutex)
	lock.Lock()

	l.data[key] = weak.Make(lock)
	runtime.AddCleanup(lock, func(key string) {
		l.mu.Lock()
		delete(l.data, key)
		l.mu.Unlock()
	}, key)

	return lock
}
