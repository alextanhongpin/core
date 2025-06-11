package lock

import "sync"

type Value[T any] struct {
	mu   sync.Mutex
	data map[string]func() T
}

func NewValue[T any]() *Value[T] {
	return &Value[T]{
		data: make(map[string]func() T),
	}
}

func (l *Value[T]) Load(key string) (func() T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	fn, ok := l.data[key]
	return fn, ok
}

func (l *Value[T]) LoadOrStore(key string, fn func() T) T {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.data[key]; !ok {
		l.data[key] = sync.OnceValue(fn)
	}

	return l.data[key]()
}
