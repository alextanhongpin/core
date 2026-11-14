# singleflight

Deduplicate work across processes with Redis: for a given key, one caller runs
the function and every other caller (in this process or any other) waits for it.

**Use it for:** preventing cache stampedes ("100 pods miss the cache at once,
only one hits the database"), or making sure an expensive refresh runs once
instead of N times.

**Don't use it for:** long-running jobs that must never overlap (use
[`dsync/lock`](../lock) and fencing), or deduplication inside a single process
only (use `golang.org/x/sync/singleflight`, no Redis needed).

```sh
go get github.com/alextanhongpin/core/dsync/singleflight
```

Requires **Go 1.27+** and **Redis 8.4+** on a single primary (not Redis Cluster).
[API reference](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/singleflight)

## Quick start

Most callers want `Cache`: a read-through cache where, on a miss, only one
caller computes the value.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alextanhongpin/core/dsync/singleflight"
	"github.com/redis/go-redis/v9"
)

type User struct {
	ID   int
	Name string
}

func loadUserFromDB(ctx context.Context, id int) (User, error) {
	return User{ID: id, Name: "Alice"}, nil // your slow query here
}

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"}) // your Redis address
	defer rdb.Close()

	users, err := singleflight.NewCache[User](rdb, singleflight.CacheConfig{})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	u, loaded, err := users.LoadOrStore(ctx, "user:42", func(ctx context.Context) (User, error) {
		return loadUserFromDB(ctx, 42)
	}, time.Minute) // cache for one minute
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(u.Name, "from cache:", loaded)
}
```

The first run prints `Alice from cache: false` (the getter ran and stored the
value). Runs within the next minute print `Alice from cache: true`, because the
value is still in Redis under `user:42`.

If you only need "run this once at a time" without a stored value, use `Group`:

```go
g, err := singleflight.New(rdb, singleflight.Config{})
// ...
did, err := g.Do(ctx, "refresh:prices", func(ctx context.Context) error {
	return refreshPrices(ctx)
})
```

`did` is true only when **this call ran `fn` and it succeeded**:

| Caller | `fn` succeeds | `fn` fails with `e` |
|--------|---------------|---------------------|
| Leader (ran `fn`) | `true, nil` | `false, e` |
| Local waiter (same `Group`) | `false, nil` | `false, e` |
| Remote waiter (other process / other `Group`) | `false, nil` | `false, nil` (failure is not visible) |

Any caller can also get `false` with `ErrTimeout`, a context error, a Redis
error, or a lease-loss error (see [Errors](#errors)). So `did == false, err == nil`
means "someone else ran it and has finished", not "it succeeded".

## API

```go
func New(client *redis.Client, cfg Config) (*Group, error)
func MustNew(client *redis.Client, cfg Config) *Group
func (*Group) Do(ctx context.Context, key string, fn func(context.Context) error) (did bool, err error)

func NewCache[T any](client *redis.Client, cfg CacheConfig) (*Cache[T], error)
func MustNewCache[T any](client *redis.Client, cfg CacheConfig) *Cache[T]
func (*Cache[T]) LoadOrStore(ctx context.Context, key string, getter func(ctx context.Context) (T, error), ttl time.Duration) (T, bool, error)

func (Config) WithDefaults() Config
func (Config) Validate() error
func (CacheConfig) WithDefaults() CacheConfig
func (CacheConfig) Validate() error

var ErrTimeout, ErrSubscriptionClosed error
const OK = "ok" // message published on the lease channel on release
```

`LoadOrStore` returns `(value, loaded, err)`. `loaded` is `false` only for the
one caller whose getter produced and stored the value; it is `true` for cache
hits, for every waiter (local or remote) that read the value the leader stored,
and for a leader that found the value already stored on its recheck. With 40
concurrent callers on a miss, expect one `loaded=false` and 39 `loaded=true`.
On error, `loaded` is `false`.

The caller keeps ownership of `client`: close it after all calls finish.

## How it works

### `Group.Do(ctx, key, fn)`

```
           same Group, same key already running?
                 │yes                      │no
                 ▼                         ▼
     wait for it, return       SET singleflight:lease:<sha256(key)> token NX PX LockTTL
     (false, leader's error)               │
                              ┌────────────┴─────────────┐
                              │ acquired (leader)        │ busy (another process leads)
                              ▼                          ▼
                     run fn, renew lease every   SUBSCRIBE to the lease key's channel
                     3/4 of LockTTL              and poll EXISTS every PollInterval
                              │                          │
                     DELEX lease (if still ours) lease gone → return (false, nil)
                     PUBLISH channel "ok"  ───►  WaitTTL passed → ErrTimeout
                     return (true, nil), or (false, err) if fn/renew/release failed
```

- The lease key is `singleflight:lease:` + SHA-256 of your key, so any key works.
- Renewal and release only act if the lease still holds this caller's token.
- Pub/Sub wakes waiters quickly; polling covers missed messages and leaders that crashed (their lease simply expires).

### `Cache.LoadOrStore(ctx, key, getter, ttl)`

1. `GET key`. Hit → JSON-decode and return `loaded=true`.
2. Miss → `Group.Do` on `key + ":fetch"`. The leader checks `GET key` again, then calls `getter`.
3. The leader stores the JSON value with a Lua script that **only writes if it
   still holds the fill lease**, so a leader that lost its lease cannot
   overwrite a newer value.
4. Everyone (leader and waiters) then reads `GET key`.

## Configuration

Zero values select the defaults. `Config` and `CacheConfig` share the first three fields.

| Field          | Default  | Meaning |
|----------------|----------|---------|
| `LockTTL`      | 10s      | Lifetime of the leader's lease; renewed every ¾ of it while `fn` runs. Minimum 1ms. |
| `WaitTTL`      | 10s      | How long a caller in another process waits for the leader before `ErrTimeout`. |
| `PollInterval` | 10ms     | How often a waiting caller checks whether the lease is gone. |
| `Suffix` (cache only) | `"fetch"` | Fill lease is named `<key>:<Suffix>`. |

`LoadOrStore`'s `ttl` argument is the cached value's lifetime: `0` = no expiry,
otherwise at least 1ms. A negative or sub-millisecond `ttl` (e.g. `500*time.Microsecond`)
returns `singleflight: ttl must be 0 (no expiry) or at least 1ms, got 500µs`
without touching Redis. An empty key or nil getter returns
`singleflight: key and getter are required`.

## Errors

Check with `errors.Is`; errors can be joined. `lock` is
`github.com/alextanhongpin/core/dsync/lock`.

| Error | When | What to do |
|-------|------|------------|
| `ErrTimeout` | Another process held the lease for longer than `WaitTTL`. | Retry later, or raise `WaitTTL` to cover your slowest `fn`. |
| `ErrSubscriptionClosed` | The Pub/Sub connection closed while waiting. | Retry. |
| `redis.Nil` (cache only) | The leader was in another process and its getter failed, so no value was stored. | Retry, or treat as a miss. |
| `lock.ErrExpired` (cache only) | The leader lost its fill lease before storing; nothing was stored. | Retry. Raise `LockTTL` if your getter is slow. |
| `lock.ErrLocked` (+ `context.Canceled`, `lock.ErrExpired`) | Lease loss in `Do`: renewal found the lease taken over, deleted or expired while `fn` ran. `fn`'s context is canceled, and `Do` returns `did=false` with a joined error, e.g. `context canceled` + `lock.ErrLocked` (renewal) + `lock.ErrExpired` (release). | `fn`'s work may have overlapped with another caller; raise `LockTTL` if `fn` stalls. |
| `context.Canceled` / `DeadlineExceeded` | Your `ctx` ended. | Normal shutdown path. |
| anything else | Error from `fn` / `getter`, a JSON error, or a Redis error. | Handle as usual. |

## Guarantees and caveats

- **Remote waiters don't get the leader's result.** In another process, `Do`
  returns `(false, nil)` once the lease is gone, even if the leader failed or
  crashed. Use `Cache` to share a value.
- **Local waiters do.** Callers on the same `Group`/`Cache` get the leader's exact error.
- **Not strict mutual exclusion.** If the leader pauses longer than `LockTTL`
  or Redis fails over, a second leader can start. `Cache` still never lets a
  leader that lost its lease store a value.
- **Cancellation is cooperative.** `fn`'s context is canceled on `ctx`
  cancellation or lease loss, but `Do` waits for `fn` to return.
- **No reentrancy.** Calling `Do`/`LoadOrStore` with the same key from inside `fn` deadlocks.
- **Single Redis primary.** The cache write touches two keys in one Lua
  script, which Redis Cluster rejects unless both hash to the same slot.
- Values are stored as JSON, so `T` must round-trip through `encoding/json`.
- The `singleflight:lease:` key prefix is reserved.

## Running the tests

Tests start a Redis 8.6 container via `dockertest`, so Docker must be running:

```sh
go test -race ./...
```
