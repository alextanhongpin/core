package lock

import "sync"

type Values[T any] struct {
	mu   sync.Mutex
	data map[string]func() (T, error)
}

func NewValues[T any]() *Values[T] {
	return &Values[T]{
		data: make(map[string]func() (T, error)),
	}
}

func (l *Values[T]) Load(key string) (func() (T, error), bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	fn, ok := l.data[key]
	return fn, ok
}

func (l *Values[T]) LoadOrStore(key string, fn func() (T, error)) (T, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.data[key]; !ok {
		l.data[key] = sync.OnceValues(fn)
	}

	return l.data[key]()
}
