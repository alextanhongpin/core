// package cmap stands for concurrent map.
package cmap

import "sync"

type ConcurrentMap[K comparable, V any] struct {
	sync.RWMutex
	data map[K]V
}

type ConcurrentMaps[K comparable, V any] []*ConcurrentMap[K, V]

func New[K comparable, V any](shards int) ConcurrentMaps[K, V] {
	return nil
}

func (c ConcurrentMaps[K, V]) Add(map[K]V) {

}

func (c ConcurrentMaps[K, V]) Set(k K, v V) {
}

func (c ConcurrentMaps[K, V]) SetNX(k K, v V) {
}

func (c ConcurrentMaps[K, V]) Get(k K) {
}
