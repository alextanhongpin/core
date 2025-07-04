package lock

import (
	"sync"
	"time"
)

type KeyedMutex struct {
	mu   sync.Mutex
	keys map[string]*keyedMutexEntry
}

type keyedMutexEntry struct {
	mutex    *sync.Mutex
	lastUsed time.Time
	refCount int
}

func NewKeyedMutex() *KeyedMutex {
	km := &KeyedMutex{
		keys: make(map[string]*keyedMutexEntry),
	}

	// Start cleanup goroutine
	go km.cleanupLoop()

	return km
}

func (l *KeyedMutex) Key(key string) sync.Locker {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.keys[key]
	if !ok {
		entry = &keyedMutexEntry{
			mutex:    new(sync.Mutex),
			lastUsed: time.Now(),
			refCount: 0,
		}
		l.keys[key] = entry
	}

	entry.lastUsed = time.Now()
	entry.refCount++

	return &keyedMutexWrapper{
		entry: entry,
		key:   key,
		km:    l,
	}
}

// cleanupLoop periodically removes unused mutexes
func (l *KeyedMutex) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanup()
	}
}

// cleanup removes mutexes that haven't been used for a while
func (l *KeyedMutex) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)

	for key, entry := range l.keys {
		if entry.refCount == 0 && entry.lastUsed.Before(cutoff) {
			delete(l.keys, key)
		}
	}
}

// Size returns the number of mutexes currently tracked
func (l *KeyedMutex) Size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.keys)
}

// keyedMutexWrapper wraps the actual mutex and manages reference counting
type keyedMutexWrapper struct {
	entry *keyedMutexEntry
	key   string
	km    *KeyedMutex
}

func (w *keyedMutexWrapper) Lock() {
	w.entry.mutex.Lock()
}

func (w *keyedMutexWrapper) Unlock() {
	w.entry.mutex.Unlock()

	// Decrement reference count
	w.km.mu.Lock()
	w.entry.refCount--
	w.km.mu.Unlock()
}
