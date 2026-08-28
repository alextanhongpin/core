# sync/cache

A concurrent, weak-reference backed cache for Go.

This package provides a generic `Cache[K comparable, V any]` that uses `sync.Map` for thread-safe storage and `weak.Pointer` (Go 1.27+) for automatic memory cleanup. Values are created lazily via a factory function and are released when no longer strongly referenced.

Inspired by https://go.dev/blog/cleanups-and-weak

## Usage

```go
import "github.com/alextanhongpin/core/sync/cache"

c := cache.New[string, *MyValue](func(key string) (*MyValue, error) {
    v, err := loadFromSource(key)
    return v, err
})

v, err := c.Get("key")
if err != nil {
    // handle error
}
```

## API

### type Cache[K comparable, V any]

```go
type Cache[K comparable, V any] struct {
    create func(K) (*V, error)
    m      sync.Map
}
```

### func New[K comparable, V any](create func(K) (*V, error)) *Cache[K, V]

Creates a new cache with the given factory function.

### func (c *Cache[K, V]) Get(key K) (*V, error)

Returns the cached value for `key`, creating it with `create` if necessary. The value is stored via `weak.Make` and automatically removed from the map when garbage collected, thanks to `runtime.AddCleanup`.

## Requirements

- Go 1.27+

## Module

```
module github.com/alextanhongpin/core/sync/cache
go 1.27.0
```
