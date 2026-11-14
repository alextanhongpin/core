# cache

Typed key/value caching on Redis or local disk, plus Redis leases so that only
one process at a time fills a missing cache entry.

**Use it when** you want a small, typed cache API over Redis; you need to stop a
"thundering herd" of processes all recomputing the same expensive value; you
want idempotency keys for API requests; or you want to memoize function
results on disk.

**Don't use it when** you need an in-memory LRU (use a dedicated library), Redis
Cluster support for the lease-based helpers, or a lock that is safe for
correctness-critical work such as payments to an external system (leases are
not fencing tokens, see [Caveats](#guarantees-and-caveats)).

```sh
go get github.com/alextanhongpin/core/dsync/cache
```

Requires Go 1.27+ and, for the Redis features, **Redis 8.4+**.
API reference: [pkg.go.dev](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/cache).

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

	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close() // you own the client; the cache never closes it

	// 1. A typed cache over Redis. Values are stored as JSON.
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

	// 2. Fill a missing entry once across ALL processes.
	lock := cache.MustNewLock(client)
	report, loaded, err := lock.LoadOrCreate(ctx, "report:daily", cache.LoadOrCreateConfig[[]byte]{
		Wait: 5 * time.Second, // wait for another process's fill instead of failing
		Create: func(ctx context.Context, key string) ([]byte, time.Duration, error) {
			return []byte("expensive result"), 10 * time.Minute, nil // value + its TTL
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s (from cache: %v)\n", report, loaded)
}
```

More runnable examples (`Func`, `Idempotent`, `AdvisoryLock`, `File`, `GoFunc`)
are in [example_test.go](example_test.go) and on pkg.go.dev.

## What's in the package

| Type / func | What it does |
|---|---|
| `NewRedis(client)` | Byte storage on Redis. Every call is one atomic Redis command. |
| `NewFile(path)` | Byte storage in one JSON file, rewritten on every write. For small local caches. |
| `NewFS(dir)` | Byte storage with one file per key. For larger values that must survive restarts. |
| `New[T](Config)` | Typed `Cache[T]` on top of any of the above, using a `Codec` (JSON default, Gob available). |
| `NewLock(client)` | Redis leases: `LoadOrCreate`, `LoadOrCreateT`, `AdvisoryLock`. |
| `Func` | Wraps `func(ctx, req) (V, ttl, error)` with a Redis read-through cache guarded by a lease. |
| `Idempotent` | Like `Func`, but rejects a key reused with a different request (`ErrConflict`). |
| `GoFunc` | Memoizes `func(ctx, req) (V, error)` in any storage, forever. |

All storages implement the same `Storage[[]byte]` interface: `Load`, `Store`,
`StoreOnce`, `LoadOrStore`, `LoadOrCreate`, `LoadAndDelete`, `CompareAndSwap`,
`CompareAndDelete`, `Exists`, `TTL`, `Expire`, `Delete`, `Close`.

**TTL rule, everywhere:** `ttl > 0` expires the key after `ttl`; `ttl <= 0`
stores it with no expiry.

## How it works

### Plain caching

`Cache[T]` encodes the value with the codec and calls the storage. `Redis`
maps each method to a single command (`GET`, `SET`, `SET NX`, `SET NX GET`,
`GETDEL`, `SET IFDEQ`, `DELEX IFDEQ`, `PTTL`, `PEXPIRE`, `DEL`), so each call
is atomic. Keys are used exactly as you pass them; add your own prefix.

`LoadOrCreate` on a storage or `Cache` only de-duplicates calls **inside one
process** (via `singleflight`). Two processes can both run `create`; the first
`SET NX` wins and the other gets the winner's value.

### Cross-process fills with `Lock`

`Lock.LoadOrCreate(ctx, key, cfg)`:

```
GET key ──hit──────────────────────────────────────────▶ return (value, loaded=true)
   │ miss
   ▼
SET cache:lease:<sha256(key)> <uuid> NX PX <Lock>
   │ taken by someone else
   ├──▶ Wait == 0 ─────────────────────────────────────▶ ErrLocked
   ├──▶ poll every 10ms until free or Wait elapses ───▶ ErrLocked (+ DeadlineExceeded)
   │ acquired
   ▼
GET key again ──hit (someone else just filled it) ─────▶ return (value, loaded=true)
   │ miss
   ▼
cfg.Create(ctx)            ◀── renewal goroutine extends the lease every
   │                           Lock*RefreshRatio (if RefreshRatio > 0);
   │                           ctx is cancelled if the lease is lost
   ▼
Lua: if GET lease == my uuid then SET key value PX ttl  ──▶ return (value, loaded=false)
     else fail ──────────────────────────────────────────▶ ErrLocked (lease lost)
   │
   ▼ always
DELEX lease IFDEQ my uuid   (release; only deletes our own lease)
```

- The lease is a normal Redis key with a TTL (`Lock`, default 10s), so a crashed
  process never blocks others for longer than that.
- With `RefreshRatio = 0` (default) there is no renewal: if `Create` runs longer
  than `Lock`, its context is cancelled and the call fails with `ErrLocked`.
  Set `RefreshRatio` (e.g. `0.5`) for work of unknown length.
- The value is written by a Lua script that first checks we still own the
  lease, so a slow process that lost its lease cannot overwrite a newer value.

`AdvisoryLock` uses the same lease around any callback (`Do`) without storing
a value. `Func` and `Idempotent` are `Lock.LoadOrCreate` plus encoding;
`Idempotent` stores `{request, response}` and compares the canonical JSON of the
request on every hit.

### Local storages

- **File** keeps everything in memory and, on every write, writes the whole map
  to a temp file, fsyncs it and renames it over `path`. It holds an exclusive
  `flock` on `path + ".lock"` while open, so a second `NewFile` on the same path
  (any process) **blocks until the first is closed**.
- **FS** stores each value in a file named by a 64-bit hash of the key, and
  expiry times in `dir/.index`. It does no cross-process locking: one `FS` per
  directory.
- Expired entries in File/FS are removed lazily when they are read.

## Configuration

`cache.Config` (for `New`):

| Field | Default | Meaning |
|---|---|---|
| `Storage` | required | `*Redis`, `*File`, `*FS` or any `Storage[[]byte]`. `Cache.Close` closes it. |
| `Codec` | `JSONCodec` | How values become bytes. `NewGobCodec()` is the alternative. |

`cache.LoadOrCreateConfig[T]` / `cache.AdvisoryLockConfig`:

| Field | Default | Meaning |
|---|---|---|
| `Create` / `Do` | required | The work. Gets a context that is cancelled if the lease is lost. `Create` returns the value and its TTL. |
| `Wait` | `0` | How long to wait if another process holds the lease. `0` = fail immediately with `ErrLocked`. |
| `Lock` | `10s` | Lease TTL. Without renewal, the work must finish within it. |
| `RefreshRatio` | `0` (off) | Renew the lease every `Lock × RefreshRatio`. Must be in `(0, 1)` when set. |
| `UseStream` | — | Deprecated, ignored. |

`cache.FuncConfig[K]` (for `Func` / `Idempotent`):

| Field | Default | Meaning |
|---|---|---|
| `Lock` | required | A `*Lock` from `NewLock`. |
| `KeyFn` | required | Builds the Redis key from the request. Include a prefix. |
| `Codec` | `JSONCodec` | Encoding of the cached result. |
| `Wait` | `0` | As above. Set it if concurrent callers for one key should share the result instead of getting `ErrLocked`. |
| `LeaseTTL` | `10s` | As `Lock` above. |
| `RefreshRatio` | `0` | As above. |

## Errors

| Error | Returned when | What to do |
|---|---|---|
| `ErrNotExist` | Key missing or expired. Also from `CompareAndSwap`/`CompareAndDelete` when the value doesn't match. | Treat as a cache miss. |
| `ErrExists` | `StoreOnce` on an existing key. | Key is already set; load it or ignore. |
| `ErrLocked` | Another process holds the lease and `Wait` is 0 or ran out; or this process lost its lease mid-work. | Retry later, serve stale data, or raise `Wait`/use `RefreshRatio`. When `Wait` ran out the error also matches `context.DeadlineExceeded`. |
| `ErrConflict` | `Idempotent` key reused with a different request. | Reject the request (HTTP 409/422). |
| `ErrClosed` | `File`/`FS` used after `Close`. | Bug in your lifecycle code. |

Errors from your callbacks, the codec and Redis are returned unchanged (use
`errors.Is`/`errors.As`). Failed callbacks are never cached.

## Guarantees and caveats

- **Leases are not fencing tokens.** After a long GC pause or network partition,
  two processes may briefly both run the callback. The value write is
  protected (Lua check), but side effects inside your callback are not.
- **Single Redis primary only** for `Lock` / `Func` / `Idempotent`: the publish
  script touches two keys (value + lease), which Redis Cluster rejects unless
  they hash to the same slot.
- **Redis 8.4+** is required (`SET IFDEQ`, `DELEX`) for `CompareAndSwap`,
  `CompareAndDelete` and everything built on `Lock`.
- **You own the Redis client.** `Redis.Close` and `Cache.Close` on a Redis
  storage do nothing to the client.
- **Close `File` and `FS`.** `File` holds a file lock until closed.
- `Cache.CompareAndSwap` compares *encoded bytes*; use a codec that encodes
  equal values identically (JSON does for structs and maps).
- `File` builds only on Unix-like systems (it uses `flock`).
- `GoFunc` keys on the function's name: different closures created at the same
  source line share cache entries.
- `TTL` returns `-1` (no expiry) or `-2` (missing) as a `time.Duration`, like
  Redis `PTTL`.

## Running the tests

Tests start a throwaway Redis 8.6 container with Docker
([redistest](https://github.com/alextanhongpin/dbtx)), so Docker must be running:

```sh
go vet ./...
go test -race -count=1 ./...
```
