package lock

import "sync"

type Lock struct {
	mu   sync.Mutex
	data map[string]*sync.Mutex
}

func New() *Lock {
	return &Lock{
		data: make(map[string]*sync.Mutex),
	}
}

func (l *Lock) Get(key string) sync.Locker {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.data[key]; !ok {
		l.data[key] = &sync.Mutex{}
	}

	return l.data[key]
}
