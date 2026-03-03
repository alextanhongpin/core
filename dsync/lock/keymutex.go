package lock

import (
	"sync"
	"sync/atomic"
)

type KeyMutex struct {
	mu   sync.Locker
	keys map[string]*KeyMutexEntry
}

type KeyMutexEntry struct {
	ref atomic.Int64
	mu  sync.Locker
}

func NewKeyMutex() *KeyMutex {
	return &KeyMutex{
		mu:   new(sync.Mutex),
		keys: make(map[string]*KeyMutexEntry),
	}
}

func (kl *KeyMutex) Lock(key string) func() {
	l := kl.entry(key)
	l.mu.Lock()

	return func() {
		kl.mu.Lock()
		if l.ref.Add(-1) == 0 {
			delete(kl.keys, key)
		}
		kl.mu.Unlock()
		l.mu.Unlock()
	}
}

func (kl *KeyMutex) entry(key string) *KeyMutexEntry {
	kl.mu.Lock()
	defer kl.mu.Unlock()

	l, ok := kl.keys[key]
	if ok {
		return l
	}

	kl.keys[key] = &KeyMutexEntry{mu: new(sync.Mutex)}
	return kl.keys[key]
}
