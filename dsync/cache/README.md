# cache

Typed key/value storage on Redis or local disk behind one interface.

**Use it when** you want a small, typed cache API over Redis, a persistent
local cache file, or to memoize function results on disk.

**Use something else when** you need an in-memory LRU (use a dedicated
library), a single fill across processes
([singleflight](../singleflight/)), a lease ([lock](../lock/)) or idempotency
keys ([idempotent](../idempotent/)).

```sh
go get github.com/alextanhongpin/core/dsync/cache
```

**Requirements:** Go 1.27+ (from `go.mod`). **Redis 8.4+** for
`Redis.CompareAndSwap` and `Redis.CompareAndDelete` (`SET IFDEQ`, `DELEX`). `File`/`FS` need no
Redis. Full reference: [pkg.go.dev](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/cache).

## Quick start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/redis/go-redis/v9"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"}) // your Redis address
	defer client.Close() // you own the client; the cache never closes it

	// A typed cache over Redis. Values are stored as JSON.
	users, err := cache.New[User](cache.Config{Storage: cache.MustNewRedis(client)})
	if err != nil {
		log.Fatal(err)
	}

	if err := users.Store(ctx, "user:1", User{ID: 1, Name: "Ada"}, time.Hour); err != nil {
		log.Fatal(err)
	}
	u, err := users.Load(ctx, "user:1")
	if errors.Is(err, cache.ErrNotExist) {
		fmt.Println("cache miss")
	} else if err != nil {
		log.Fatal(err)
	}
	fmt.Println(u.Name)
}
```

It prints `Ada`. The key stays in Redis for an hour.

## API

TTL rule for every store/create call: `ttl > 0` expires the key after `ttl`;
`ttl <= 0` stores it with no expiry. (Exception: `Expire` with `ttl <= 0`
deletes the key.) Keys are used exactly as you pass them; add your own prefix.

### Storages and the typed cache

```go
func NewRedis(client *redis.Client) (*Redis, error) // one atomic Redis command per call
func NewFile(path string) (*File, error)            // all entries in one JSON file; Unix only
func NewFS(dir string) (*FS, error)                 // one file per key in dir
func New[T any](cfg Config) (*Cache[T], error)      // typed values on any Storage[[]byte]
// MustNewRedis, MustNew[T] have the same parameters and panic instead of returning an error.
```

`*Redis`, `*File`, `*FS` implement `Storage[[]byte]`; `*Cache[T]` implements
`Storage[T]`. The methods (shown for `Storage[T]`):

```go
Load(ctx context.Context, key string) (T, error)                                   // ErrNotExist on miss
Store(ctx context.Context, key string, value T, ttl time.Duration) error
StoreOnce(ctx context.Context, key string, value T, ttl time.Duration) error        // ErrExists if set
LoadOrStore(ctx context.Context, key string, value T, ttl time.Duration) (curr T, loaded bool, err error)
LoadOrCreate(ctx context.Context, key string,
	fn func(ctx context.Context, key string) (T, time.Duration, error)) (curr T, loaded bool, err error)
LoadAndDelete(ctx context.Context, key string) (T, error)
CompareAndSwap(ctx context.Context, key string, old, value T, ttl time.Duration) error // ErrNotExist if missing or != old
CompareAndDelete(ctx context.Context, key string, old T) error                         // ErrNotExist if missing or != old
Exists(ctx context.Context, key string) (bool, error)
TTL(ctx context.Context, key string) (time.Duration, error)  // -1 = no expiry, -2 = missing (as time.Duration, like PTTL)
Expire(ctx context.Context, key string, ttl time.Duration) error // ErrNotExist if missing
Delete(ctx context.Context, keys ...string) (int64, error)       // number of keys that existed
Close() error
```

`loaded` is `true` when the value already existed and `false` when this call
stored it. `File` and `FS` also have `Size(ctx context.Context) (int, error)`.

### Codecs

`Codec` turns values into bytes. Both codecs are empty structs, so either form works:

```go
cache.Config{Storage: s}                               // JSON (default when Codec is nil)
cache.Config{Storage: s, Codec: cache.NewJSONCodec()}  // or &cache.JSONCodec{} / cache.JSONCodec{}
cache.Config{Storage: s, Codec: cache.NewGobCodec()}   // or cache.GobCodec{}; Go-only bytes, allows non-string map keys
```

### GoFunc

```go
func GoFunc[K, V any](fn func(context.Context, K) (V, error),
	c Storage[[]byte], codec Codec) func(context.Context, K) (V, error) // nil codec = JSON; never expires
```

`GoFunc` memoizes on disk (or any storage); the key is fn's name plus the JSON
of the argument:

```go
fs, err := cache.NewFS("./.cache")
if err != nil {
	log.Fatal(err)
}
defer fs.Close()
square := cache.GoFunc(func(ctx context.Context, n int) (int, error) { return n * n, nil }, fs, nil)
v, err := square(ctx, 4) // 16; later calls (and later runs) read it from ./.cache
```

More runnable examples are in [example_test.go](example_test.go).

## Configuration

`cache.Config` (for `New`):

| Field | Type | Default | Meaning |
|---|---|---|---|
| `Storage` | `Storage[[]byte]` | required | `*Redis`, `*File`, `*FS` or your own. `Cache.Close` closes it. |
| `Codec` | `Codec` | JSON | See [Codecs](#codecs). |

## How it works

### Plain caching

`Cache[T]` encodes the value with the codec and calls the storage. `Redis`
maps each method to a single command (`GET`, `SET`, `SET NX`, `SET NX GET`,
`GETDEL`, `SET IFDEQ`, `DELEX IFDEQ`, `PTTL`, `PEXPIRE`, `DEL`), so each call
is atomic.

`LoadOrCreate` on a storage or `Cache` only de-duplicates calls **inside one
process** (via `golang.org/x/sync/singleflight`). Two processes can both run
`create`; the first `SET NX` wins and the other gets the winner's value. Use
[singleflight](../singleflight/)'s `Cache` when `create` must run once across
processes.

### Local storages

- **File** keeps everything in memory and, on every write, writes the whole map
  to a temp file, fsyncs it and renames it over `path`. It holds an exclusive
  `flock` on `path + ".lock"` while open, so a second `NewFile` on the same path
  (any process) **blocks until the first is closed**.
- **FS** stores each value in a file named by a 64-bit hash of the key, and
  expiry times in `dir/.index`. Each file is replaced atomically (temp file,
  fsync, rename), so a crash never leaves a truncated value. It does no
  cross-process locking: one `FS` per directory.
- Expired entries in File/FS are removed lazily when they are read.

## Errors

| Error | Returned when | What to do |
|---|---|---|
| `ErrNotExist` | Key missing or expired. Also from `CompareAndSwap`/`CompareAndDelete` when the value doesn't match. | Treat as a cache miss. |
| `ErrExists` | `StoreOnce` on an existing key. | Key is already set; load it or ignore. |
| `ErrClosed` | `File`/`FS` used after `Close`. | Bug in your lifecycle code. |

Errors from your callbacks, the codec and Redis are returned unchanged (use
`errors.Is`/`errors.As`). Failed `create` calls are never cached.

## Guarantees and caveats

- **You own the Redis client.** `Redis.Close` and `Cache.Close` on a Redis
  storage do nothing to the client.
- **Close `File` and `FS`.** `File` holds a file lock until closed.
- `Cache.CompareAndSwap` compares *encoded bytes*; use a codec that encodes
  equal values identically (JSON does for structs and maps).
- `GoFunc` keys on the function's name: different closures created at the same
  source line share cache entries.

## Running the tests

Tests start a throwaway Redis 8.6 container with Docker
([redistest](https://github.com/alextanhongpin/dbtx)), so Docker must be running:

```sh
go vet ./...
go test -race -count=1 ./...
```
