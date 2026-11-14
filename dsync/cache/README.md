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

**Requirements:** Go 1.27+ (from `go.mod`). **Redis 8.4+** for the Redis
features (`SET IFDEQ` and `DELEX` are used by `CompareAndSwap`,
`CompareAndDelete` and everything built on `Lock`). `File`/`FS` need no Redis.
Full reference: [pkg.go.dev](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/cache).

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

Expected output on the first run:

```
Ada
expensive result (from cache: false)
```

The keys stay in Redis (`user:1` for 1h, `report:daily` for 10m), so a second
run within 10 minutes prints `(from cache: true)` and `Create` does not run.
Delete the keys (`DEL user:1 report:daily`) to start over.

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

### Lock (Redis leases)

```go
func NewLock(client *redis.Client) (*Lock, error) // MustNewLock panics instead
func (*Lock) LoadOrCreate(ctx context.Context, key string, cfg LoadOrCreateConfig[[]byte]) (curr []byte, loaded bool, err error)
func (*Lock) LoadOrCreateT[T any](ctx context.Context, key string, cfg LoadOrCreateConfig[T]) (curr T, loaded bool, err error) // JSON-encoded
func (*Lock) AdvisoryLock(ctx context.Context, key string, cfg AdvisoryLockConfig) error // returns Do's error

// Callbacks:
LoadOrCreateConfig[T].Create func(ctx context.Context, key string) (T, time.Duration, error) // value + its TTL
AdvisoryLockConfig.Do        func(ctx context.Context, key string, token []byte) error
```

`Do` receives the key you passed and the lease token (a random ID identifying
this holder; most callers ignore it). Its `ctx` is cancelled if the lease is lost.

```go
err := lock.AdvisoryLock(ctx, "job:cleanup", cache.AdvisoryLockConfig{
	RefreshRatio: 0.5, // renew the 10s lease every 5s while Do runs
	Do: func(ctx context.Context, key string, token []byte) error {
		return runCleanup(ctx)
	},
})
if errors.Is(err, cache.ErrLocked) {
	// another process is running it
}
```

### Func, Idempotent, GoFunc

The function comes first, the config second. `K` (request) and `V` (result)
are inferred from `fn`; `fn` returns the result **and the TTL to cache it for**.

```go
func Func[K, V any](fn func(ctx context.Context, req K) (V, time.Duration, error),
	cfg FuncConfig[K]) (func(ctx context.Context, req K) (V, bool, error), error)
func Idempotent[K, V any](fn func(ctx context.Context, req K) (V, time.Duration, error),
	cfg FuncConfig[K]) (func(ctx context.Context, req K) (V, bool, error), error)
// MustFunc, MustIdempotent: same parameters, return only the wrapped func, panic on error.

func GoFunc[K, V any](fn func(context.Context, K) (V, error),
	c Storage[[]byte], codec Codec) func(context.Context, K) (V, error) // nil codec = JSON; never expires

FuncConfig[K].KeyFn func(ctx context.Context, req K) (string, error) // required
```

The returned function's `bool` is `loaded` (true = served from cache).

```go
getUser := cache.MustFunc(func(ctx context.Context, id int) (string, time.Duration, error) {
	return fmt.Sprintf("user-%d", id), time.Minute, nil // result, cache TTL, error
}, cache.FuncConfig[int]{
	Lock:  cache.MustNewLock(client),
	KeyFn: func(ctx context.Context, id int) (string, error) { return fmt.Sprintf("user:%d", id), nil },
	Wait:  5 * time.Second,
})
name, loaded, err := getUser(ctx, 7)
```

**Idempotent:** the idempotency key is whatever your `KeyFn` returns, so put
the client-supplied key (e.g. the `Idempotency-Key` header) in the request
type. The first response for a key is stored with the request and replayed on
repeats; the same key with a different request (compared as canonical JSON)
returns `ErrConflict`.

```go
type Payment struct {
	Key    string // idempotency key from the client
	Amount int
}

pay := cache.MustIdempotent(func(ctx context.Context, p Payment) (string, time.Duration, error) {
	return fmt.Sprintf("charged %d", p.Amount), 24 * time.Hour, nil // keep the key for 24h
}, cache.FuncConfig[Payment]{
	Lock:  cache.MustNewLock(client),
	KeyFn: func(ctx context.Context, p Payment) (string, error) { return "payment:" + p.Key, nil },
})

pay(ctx, Payment{Key: "abc", Amount: 100}) // "charged 100", false, nil  (runs fn)
pay(ctx, Payment{Key: "abc", Amount: 100}) // "charged 100", true,  nil  (replayed)
pay(ctx, Payment{Key: "abc", Amount: 999}) // "",            false, ErrConflict
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

> **`Lock` means different things in different configs.** In
> `LoadOrCreateConfig` and `AdvisoryLockConfig`, `Lock` is the lease **TTL**
> (`time.Duration`). In `FuncConfig`, `Lock` is the **`*Lock`** from
> `NewLock`, and the lease TTL is called `LeaseTTL`.

`cache.Config` (for `New`):

| Field | Type | Default | Meaning |
|---|---|---|---|
| `Storage` | `Storage[[]byte]` | required | `*Redis`, `*File`, `*FS` or your own. `Cache.Close` closes it. |
| `Codec` | `Codec` | JSON | See [Codecs](#codecs). |

`cache.LoadOrCreateConfig[T]` / `cache.AdvisoryLockConfig`:

| Field | Type | Default | Meaning |
|---|---|---|---|
| `Create` / `Do` | func, see [Lock](#lock-redis-leases) | required | The work. Its ctx is cancelled if the lease is lost. |
| `Wait` | `time.Duration` | `0` | How long to wait if another process holds the lease. `0` = fail immediately with `ErrLocked`. |
| `Lock` | `time.Duration` | `10s` | Lease TTL. Without renewal, the work must finish within it. |
| `RefreshRatio` | `float32` | `0` (off) | Renew the lease every `Lock × RefreshRatio`. Must be in `(0, 1)` when set. |
| `UseStream` | `bool` | — | Deprecated, ignored. |

`cache.FuncConfig[K]` (for `Func` / `Idempotent`):

| Field | Type | Default | Meaning |
|---|---|---|---|
| `Lock` | `*Lock` | required | From `NewLock`. **Not** a TTL. |
| `KeyFn` | `func(ctx context.Context, req K) (string, error)` | required | Builds the Redis key from the request. Include a prefix. |
| `Codec` | `Codec` | JSON | Encoding of the cached result. |
| `Wait` | `time.Duration` | `0` | As above. Set it if concurrent callers for one key should share the result instead of getting `ErrLocked`. |
| `LeaseTTL` | `time.Duration` | `10s` | Lease TTL (what `Lock` means in the configs above). |
| `RefreshRatio` | `float32` | `0` | As above. |

## How it works

### Plain caching

`Cache[T]` encodes the value with the codec and calls the storage. `Redis`
maps each method to a single command (`GET`, `SET`, `SET NX`, `SET NX GET`,
`GETDEL`, `SET IFDEQ`, `DELEX IFDEQ`, `PTTL`, `PEXPIRE`, `DEL`), so each call
is atomic.

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
