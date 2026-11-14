# lock

Run a function while holding a Redis lease on a key, so only one process at a
time does the work for that key.

**Use it for:** cron jobs running on several replicas, cache rebuilds,
migrations, "only one worker processes order 42 at a time".

**Don't use it for:** strict mutual exclusion where an overlap would corrupt
data (a lease is not a fencing token, see [Caveats](#guarantees-and-caveats)),
or for coalescing callers that all need the *result* (use
[`dsync/singleflight`](../singleflight)).

```sh
go get github.com/alextanhongpin/core/dsync/lock
```

Requires **Go 1.27+** and **Redis 8.4+** (uses `DELEX` and `SET ... IFDEQ`).
[API reference](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/lock)

## Quick start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"}) // your Redis address
	defer rdb.Close()

	locker, err := lock.New(lock.NewClient(rdb), lock.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	err = locker.Do(context.Background(), "reports:daily", func(ctx context.Context) error {
		// Only one process runs this at a time.
		// Stop when ctx is canceled: the lease was lost or the caller gave up.
		fmt.Println("rebuilding report")
		return nil
	})
	if errors.Is(err, lock.ErrLocked) {
		log.Println("another process is still running it; skipping")
		return
	}
	if err != nil {
		log.Fatal(err)
	}
}
```

It prints `rebuilding report` and exits 0. Running it again prints the same:
`Do` deletes the key when `fn` returns, so nothing persists in Redis. If a
previous run crashed while holding the lease, the next run waits (see
`DefaultRetry` below) until the key expires after `LockTTL`, or gives up with
`ErrLocked`.

## API

```go
func New(c Backend, cfg Config) (*Locker, error)
func MustNew(c Backend, cfg Config) *Locker // panics on invalid cfg
func NewClient(client *redis.Client) *Client // Redis Backend; you still Close client

func (*Locker) Do(ctx context.Context, key string, fn func(ctx context.Context) error) error

// Raw lease operations (promoted from the Backend). You choose the token.
func (*Locker) Lock(ctx context.Context, key, token string, ttl time.Duration) error   // ErrLocked if key exists
func (*Locker) Extend(ctx context.Context, key, token string, ttl time.Duration) error // ErrLocked if token no longer owns key
func (*Locker) Unlock(ctx context.Context, key, token string) error                    // ErrExpired if token no longer owns key

func Func[K, V any](
	fn func(ctx context.Context, req K) (V, error),
	l Doer,
	keyFn func(ctx context.Context, req K) (string, error),
) func(ctx context.Context, req K) (V, error)

func DefaultConfig() Config
func DefaultRetry() Retry
func NoRetry() Retry // single attempt: fail fast with ErrLocked
func (Config) WithDefaults() Config
func (Config) Validate() error

type Backend interface { // implemented by *Client; Locker embeds it
	Lock(ctx context.Context, key, token string, ttl time.Duration) error
	Extend(ctx context.Context, key, token string, ttl time.Duration) error
	Unlock(ctx context.Context, key, token string) error
}
type Doer interface {
	Do(ctx context.Context, key string, fn func(ctx context.Context) error) error
}
type Retry interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}
```

**`Func`** runs each call under the key that `keyFn` returns for that request.
If `keyFn` errors, `fn` is not called. The result of `fn` is returned even
when the error is non-nil (e.g. `fn` succeeded but release failed).

```go
chargeOrder := func(ctx context.Context, orderID int) (string, error) {
	return fmt.Sprintf("charged order %d", orderID), nil
}
charge := lock.Func(chargeOrder, locker, func(_ context.Context, orderID int) (string, error) {
	return fmt.Sprintf("order:%d", orderID), nil
})
res, err := charge(ctx, 42) // "charged order 42", nil
```

**Raw `Lock`/`Extend`/`Unlock`** do no retry, renewal or local queueing.
The token identifies the owner: pass a value unique to this holder (e.g.
`uuid.NewV7().String()` from the standard library's `uuid` package) and the
same value to `Extend` and `Unlock`. (`Do` generates its own token for you;
that is the UUIDv7 in [How it works](#how-it-works).)

```go
tok := uuid.NewV7().String()
if err := locker.Lock(ctx, "jobs:sync", tok, 10*time.Second); err != nil {
	return err // lock.ErrLocked if someone else holds it
}
defer locker.Unlock(context.WithoutCancel(ctx), "jobs:sync", tok)
// ... work; call locker.Extend(ctx, "jobs:sync", tok, 10*time.Second) before the TTL runs out
```

## How it works

The lease is a single Redis string: key = your key, value = a random token
(UUIDv7) unique to this `Do` call, with a TTL.

```
Do(ctx, key, fn)
 │
 ├─ wait for other Do calls on the same key in this process (local queue)
 │
 ├─ acquire:  SET key token NX PX LockTTL      ── busy? retry via Config.Retry
 │
 ├─ run fn(ctx') ───────────────────────────────┐
 │    every LockTTL*RefreshRatio:               │ ctx' is canceled if:
 │    SET key token IFDEQ digest(token) PX TTL  │  - ctx is canceled
 │    (renew only if we still own it)           │  - renewal fails
 │                                              │  - LockTTL passes (no renewal)
 ├─ wait for fn to return ◄─────────────────────┘
 │
 └─ release:  DELEX key IFDEQ digest(token)    (delete only if we still own it)
```

- Renew and release compare the stored token, so a process can never extend
  or delete a lease that someone else now holds.
- Release runs even if `fn` panics or `ctx` is canceled (with its own 5 s timeout).
- If acquiring fails ambiguously (e.g. a timeout after `SET` was sent), `Do`
  deletes its own token so the key does not stay blocked until it expires.

## Configuration

`lock.DefaultConfig()` is the recommended starting point. With a `Config{}`
literal, zero fields get the defaults below; note that **`Config{}` has renewal
off**.

| Field          | `DefaultConfig()` | Zero value means | Meaning |
|----------------|-------------------|------------------|---------|
| `LockTTL`      | 30s               | 30s              | Lifetime of the Redis key. Minimum 1ms. |
| `RefreshRatio` | 0.8 (renew every 24s) | renewal off  | Renew every `LockTTL × RefreshRatio`. Must be in `[0, 1)`. With 0, `fn`'s context hits its deadline after `LockTTL` (cause `ErrLockTimeout`). |
| `Retry`        | `DefaultRetry()`  | `DefaultRetry()` | How to wait for a busy key. `DefaultRetry` retries up to 10 times; retry *n* sleeps a random time in `[0, min(50ms·2ⁿ, 2s))` (full jitter), so it gives up after **at most ~13.1s, ~6.5s on average**. `NoRetry()` tries once. Any `*retry.Retry` from `core/sync/retry` works. |
| `Logger`       | –                 | –                | Deprecated, ignored. |

## Errors

Errors from `Do` may be joined (e.g. your error plus a release error); always
check with `errors.Is`.

**Inside `fn`**, `ctx.Err()` only says *that* the context ended
(`context.DeadlineExceeded` or `context.Canceled`); use `context.Cause(ctx)`
to see *why*:

| Reason | `ctx.Err()` | `context.Cause(ctx)` |
|--------|-------------|----------------------|
| Renewal off, `LockTTL` passed | `context.DeadlineExceeded` | `ErrLockTimeout` |
| Renewal found the key gone or taken | `context.Canceled` | matches `ErrLocked` |
| Renewal could not finish before the lease expired | `context.Canceled` | matches `ErrExpired` |
| Your `ctx` ended | whatever your ctx reports | your ctx's cause |

`Do`'s returned error includes that cause, so `errors.Is(err, lock.ErrLockTimeout)`
etc. work on it directly.

| Error | When | What to do |
|-------|------|------------|
| `ErrLocked` | Key is held by someone else (after retries), or the key was deleted/taken over during renewal. | Skip, or try later. With `DefaultRetry` the error also matches `retry.ErrLimitExceeded`. |
| `ErrExpired` | The lease was gone at release time, or renewal could not finish before the lease expired. | Your work may have overlapped with another holder; make it idempotent or verify it. |
| `ErrLockTimeout` | Renewal is off and `fn` ran longer than `LockTTL`. | Raise `LockTTL` or enable `RefreshRatio`. |
| `context.Canceled` / `DeadlineExceeded` | Your `ctx` ended. | Normal shutdown path. |
| anything else | Error returned by `fn`, or a Redis error. | Handle as usual. |

## Guarantees and caveats

- **Not fencing.** If your process pauses (GC, VM freeze) longer than the
  TTL, or Redis fails over before replicating the key, two holders can
  overlap. If that would be harmful, also guard the resource itself (e.g. a
  version column or unique constraint).
- **Cancellation is cooperative.** `Do` cancels `fn`'s context but waits for
  `fn` to return. A callback that ignores `ctx` keeps running after the lease is lost.
- **No reentrancy.** Calling `Do` with the same key from inside `fn` deadlocks.
- **One renewal failure stops the work.** A single failed renew (including a
  transient network error) cancels `fn`; choose a TTL that tolerates your
  Redis latency.
- **Single Redis primary.** `NewClient` takes a `*redis.Client`. Each lease
  is one key, so other topologies work if you implement `Backend` yourself.
- Same-process callers on the same key queue locally on one `Locker`, so they
  don't all poll Redis. Use one `Locker` per process.

## Running the tests

Tests start a throwaway Redis in Docker (via `dockertest`), so Docker must be running:

```sh
go test -race ./...
```
