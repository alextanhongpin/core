package cmap

import (
	"time"
)

type ttlMap[K comparable, V any] struct {
	data map[K]*value[V]
}

func (t *ttlMap[K, V]) store(k K, v V, ttl time.Duration) {
	t.data[k] = &value[V]{
		data:     v,
		deadline: time.Now().Add(ttl),
	}
}

func (t *ttlMap[K, V]) loadMany(ks ...K) map[K]V {
	m := make(map[K]V)

	for _, k := range ks {
		v, ok := t.load(k)
		if !ok {
			continue
		}

		m[k] = v
	}

	return m
}

func (t *ttlMap[K, V]) load(k K) (V, bool) {
	v, ok := t.data[k]
	if ok && v.expired() {
		delete(t.data, k)

		return v.data, false
	}

	return v.data, ok
}

type value[V any] struct {
	data     V
	deadline time.Time
}

func (v *value[V]) expired() bool {
	return gte(time.Now(), v.deadline)
}

func gte(a, b time.Time) bool {
	return !a.Before(b)
}
