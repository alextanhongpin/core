# cache

A concurrent, generic cache that holds its values through weak references and
creates missing values lazily.

**Use it for:** deduplicating shared, expensive-to-build objects (open files,
in-flight futures) for as long as someone still uses them.

**Don't use it for:** durable caching. Values are reclaimed by the garbage
collector as soon as nothing else references them. Use a regular map or an LRU
if entries must survive. For distributed caching see `dsync/cache`.

`sync/dataloader`, `sync/promise` and `dsync/lock` are built on this package, so
its API is kept stable.

## Installation

```sh
go get github.com/alextanhongpin/core/sync/cache
```

Requires Go 1.27+ (`weak`, `runtime.AddCleanup`).

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/sync/cache"
)

func main() {
	c := cache.New(func(key string) (*string, error) {
		v := "value for " + key
		return &v, nil
	})

	v, loaded, err := c.LoadOrCreate("a")
	fmt.Println(*v, loaded, err) // value for a false <nil>

	v2, loaded, _ := c.LoadOrCreate("a")
	fmt.Println(v == v2, loaded) // true true (v is still referenced)
}
```

## API

```go
func New[K comparable, V any](create func(K) (*V, error)) *Cache[K, V]

// Returns the live value for key; loaded is true if it already existed.
func (*Cache[K, V]) LoadOrCreate(key K) (value *V, loaded bool, err error)
```

## Behavior and guarantees

- **Weak references.** The cache never keeps a value alive. Retain the returned
  pointer for as long as you need it; once unreferenced it may be reclaimed at
  any GC cycle and the next `LoadOrCreate` calls `create` again.
- **`create` may run more than once per key.** Concurrent misses can each call
  `create`; one result is installed and the rest are dropped (losers receive the
  winner with `loaded == true`). `create` must be safe to repeat, and anything it
  allocates must tolerate being discarded. Re-creation after GC also calls it
  again.
- **Errors.** If `create` returns an error or a nil `*V`, nothing is stored and
  the error is returned (`cache: create returned nil value` for nil).
- **Concurrency.** Safe for concurrent use (`sync.Map` underneath). There is no
  delete or size API; entries disappear only through GC.
- **Cleanup.** A `runtime.AddCleanup` on each installed value removes its map
  entry once reclaimed; stale entries found earlier are deleted eagerly.

## Testing

```sh
go vet ./... && go test -race -count=1 ./...
```
