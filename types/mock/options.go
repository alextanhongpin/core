package mock

import "maps"

// Map[T any] is a type-safe map.
type Map[T any] map[string][]T

func (m Map[T]) Values(key string) []T {
	return m[key]
}

func (m Map[T]) With(key string, vals ...T) Map[T] {
	cloned := maps.Clone(m)
	cloned[key] = append(cloned[key], vals...)
	return cloned
}

func (m Map[T]) Del(key string) {
	delete(m, key)
}

type Options = Map[string]

type Calls = Map[[]any]
