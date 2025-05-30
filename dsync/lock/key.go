package lock

import "sync"

type KeyedMutex struct {
	mu   sync.Mutex
	keys map[string]*sync.Mutex
}

func NewKeyedMutex() *KeyedMutex {
	return &KeyedMutex{
		keys: make(map[string]*sync.Mutex),
	}
}

func (l *KeyedMutex) Key(key string) sync.Locker {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.keys[key]; !ok {
		l.keys[key] = new(sync.Mutex)
	}

	return l.keys[key]
}
