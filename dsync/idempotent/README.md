# idempotent

Run a function **once per idempotency key** and replay its stored response to every retry, using Redis.

Use it when the same request can arrive more than once and running it twice would be harmful:

- HTTP APIs that accept an `Idempotency-Key` header (payments, orders, sign-ups).
- Queue consumers that may receive the same message twice.

Don't use it for:

- Caching reads. Use a cache.
- Strict exactly-once side effects. The function can run again after a crash, after Redis loses data, or once `KeepTTL` has passed (see [Caveats](#guarantees--caveats)).
- Mutual exclusion without storing a result. Use a lock.

```sh
go get github.com/alextanhongpin/core/dsync/idempotent
```

Requires **Redis 8.4+** (the package uses `SET ... IFDEQ` and `DELEX`). API reference: [pkg.go.dev](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/idempotent).

## Quick start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/alextanhongpin/core/dsync/idempotent"
	"github.com/redis/go-redis/v9"
)

type ChargeRequest struct {
	AccountID string `json:"account_id"`
	Cents     int64  `json:"cents"`
}

type Receipt struct {
	ID string `json:"id"`
}

func main() {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	// One Idempotent per process. It is safe for concurrent use.
	idem, err := idempotent.NewWithRedis(rdb)
	if err != nil {
		log.Fatal(err)
	}

	// Wrap the function that must not run twice.
	// The zero HandlerConfig uses the defaults (LockTTL=10s, KeepTTL=24h).
	charge, err := idem.HandlerFunc(func(ctx context.Context, req ChargeRequest) (Receipt, error) {
		fmt.Println("charging", req.AccountID, req.Cents)
		return Receipt{ID: "rcpt-123"}, nil
	}, idempotent.HandlerConfig{})
	if err != nil {
		log.Fatal(err)
	}

	// Put the tenant and the operation in the key, then the client-supplied key.
	key := "tenant-1:charge:" + "client-key-abc"
	req := ChargeRequest{AccountID: "acc-1", Cents: 500}

	for range 2 {
		receipt, shared, err := charge.Do(ctx, key, req)
		switch {
		case errors.Is(err, idempotent.ErrRequestMismatch):
			log.Fatal("key reused with a different request")
		case errors.Is(err, idempotent.ErrRequestInFlight):
			log.Fatal("still running elsewhere, retry later")
		case err != nil:
			log.Fatal(err)
		}
		// The first call prints "charging" and shared=false.
		// The second call returns the stored receipt with shared=true.
		fmt.Println(receipt.ID, "shared:", shared)
	}
}
```

`Do` returns `(response, shared, err)`. `shared` is `true` when the response was replayed from Redis instead of produced by this call.

Need a struct instead of a function? Implement `idempotent.Task[K, V]` (a `Do(ctx, req K) (V, error)` method) and pass it to `idem.Handler`. Each constructor has a `Must...` variant that panics, for use at startup.

## How it works

Each key is one Redis string that holds a JSON document. The document is either **in flight** or **done**:

```
in flight: {"done":false, "token":"<uuid v7>", "request":{...}, "response":<zero>}   TTL = LockTTL
done:      {"done":true,  "request":{...},     "response":{...}}                   TTL = KeepTTL
```

What `Do(ctx, key, req)` does:

```
          same-process callers for this key queue up here (one at a time)
                                   │
              SET key <in-flight> NX GET PX LockTTL
                 │                                  │
           key was absent                     key existed
                 │                                  │
     run fn(ctx, req)                  ┌────────────┼─────────────────┐
     renew TTL every 0.7×LockTTL       │            │                 │
                 │                   in flight   done, same req   done, other req
        ┌────────┴────────┐            │            │                 │
     fn ok             fn error      ErrRequest   (stored response,  ErrRequest
        │              or panic      InFlight      shared=true)      Mismatch
  swap in-flight       delete key
  for done entry       (so a retry
  (TTL = KeepTTL)       can run fn)
```

Details:

- **Ownership check.** Renewing, committing and deleting are compare-and-set operations (`SET IFDEQ` / `DELEX IFDEQ`) that only succeed while the key still holds *this* caller's in-flight document. The document includes a random token, so two in-flight documents never look the same. If the check fails, the caller has lost the key, and it gets `ErrLockConflict` instead of overwriting someone else's entry.
- **Lease renewal.** While `fn` runs, a background goroutine renews the in-flight TTL every 70% of `LockTTL`. So `fn` may run longer than `LockTTL`; `LockTTL` only limits how long a crashed process can block the key. If a renewal fails, `fn`'s context is canceled and its result is thrown away.
- **Same process: callers wait.** Callers for the same key in one process take turns. The second caller waits for the first one to finish, then reads the stored result. A caller whose context is canceled stops waiting.
- **Other processes: no waiting.** A caller in another process that finds an in-flight entry gets `ErrRequestInFlight` straight away.
- **Comparing requests.** The incoming request is encoded with `encoding/json`, and the bytes are compared with the stored request bytes.
- **Caller cancellation.** If the caller's context is canceled while `fn` is still running, the lease is still renewed until `fn` returns. If `fn` then succeeds, its result is stored and returned, so a retry does not repeat the side effect. If `fn` returns an error, the key is deleted.
- **Keys.** Keys are used exactly as given (no prefix is added), so namespace them yourself, e.g. `"<tenant>:<operation>:<client key>"`.

## Configuration

`HandlerConfig` is passed when you create a handler. Fields left at zero use the default.

| Field     | Default | Meaning |
|-----------|---------|---------|
| `LockTTL` | `10s`   | TTL of the in-flight entry. It is renewed every `0.7 × LockTTL` while `fn` runs, so it only limits how long a crashed process blocks the key. Minimum `1ms`. |
| `KeepTTL` | `24h`   | How long the completed response is stored and replayed. After it expires, the same key runs `fn` again. Minimum `1ms`. |

`DefaultConfig()` returns the defaults. `WithDefaults()` fills in zero fields, and `Validate()` checks the minimums. The constructors call both for you.

## Errors

| Error | Meaning | What to do (HTTP) |
|-------|---------|-------------------|
| `ErrEmptyKey` | `key` was `""`. | `400 Bad Request`: the idempotency key is missing. |
| `ErrRequestMismatch` | The key already has a stored result, but it was produced by a different request. | `422 Unprocessable Entity`: the client reused a key with a different body. |
| `ErrRequestInFlight` | Another process is still running `fn` for this key. | `409 Conflict` (or `429` with `Retry-After`): retry later. |
| `ErrLockConflict` | This caller lost the key (the lease expired or the entry changed). Nothing was stored. | `500`, or retry. Raise `LockTTL` if this happens often. |
| errors from `fn` | Passed through (wrapped, so use `errors.Is`/`errors.As`). Nothing is stored, so a retry runs `fn` again. | Your own mapping. |
| `context.Canceled` / `DeadlineExceeded` | The caller's context ended before `fn` started or while it was waiting its turn. | Usually nothing to do. |
| Redis / JSON errors | Wrapped and returned. | `500`. |

`ErrFunctionExecutionFailed` is deprecated and never returned.

## Guarantees & caveats

- **At most once while the result is stored, not exactly once.** `fn` can run again if: the process crashes mid-run (after `LockTTL`), Redis loses the key (failover, eviction, restart without persistence), or `KeepTTL` expires. If a duplicate is unacceptable, make the side effect itself idempotent too (for example, send the key to your payment provider).
- **Errors are not stored.** Only successful results are kept, so a failed request can be retried with the same key.
- **JSON round-trip.** Requests and responses are stored as JSON. Unexported fields are dropped, and a response of type `any` comes back as `map[string]any` / `float64`. Use concrete, JSON-friendly types.
- **Request comparison is byte-for-byte JSON.** Field order follows the struct definition, and map keys are sorted. If you change a request type's fields or JSON tags, requests that already have stored results will report `ErrRequestMismatch` until `KeepTTL` passes.
- **In-flight entries are not checked for a mismatch.** A different request that arrives while the first one is still running gets `ErrRequestInFlight`, not `ErrRequestMismatch`.
- **Don't call the same key from inside `fn`.** It deadlocks on the per-key queue. Calls for different keys are fine.
- **Make `fn` respect `ctx`.** Its context is canceled when the caller cancels or the lease is lost.
- **Panics** in `fn` delete the in-flight entry and then propagate to the caller.
- **Single Redis primary.** `NewWithRedis` takes a `*redis.Client`. Behaviour across replica failover follows Redis's own (asynchronous) replication.
- After `fn` returns, committing the result or deleting the key uses its own 5-second timeout, separate from the caller's context, so cleanup still runs after the caller cancels.

## Testing

Tests start a throwaway Redis in Docker (via `dockertest`), so you need a running Docker daemon:

```sh
cd dsync/idempotent
go test -race ./...
```

For unit tests that should not need Redis, pass your own `idempotent.Store` implementation to `idempotent.New`.
