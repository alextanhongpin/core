# circuitbreaker

A circuit breaker whose state is stored in Redis. When a dependency starts failing, **every** instance of your service stops calling it at the same time, and they all recover together.

**Use it when** several replicas call the same dependency (a payments API, a database, another microservice) and should back off together instead of each one finding out on its own.
**Don't use it when** you run a single process (an in-memory breaker is faster and needs no Redis), or when you need a hard limit on concurrent probes. Half-open does not limit concurrency, see [caveats](#guarantees-and-caveats).

## Install

```sh
go get github.com/alextanhongpin/core/dsync/circuitbreaker
```

Requires **Redis 7.4 or newer**. It uses Redis Functions and hash-field expiration (`HPEXPIRE`).

## Quick start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()

	// Install the Lua functions. This is safe to call on every start.
	if err := circuitbreaker.Setup(ctx, client); err != nil {
		log.Fatal(err)
	}

	cb, err := circuitbreaker.New(client, circuitbreaker.Config{
		FailureThreshold: 5,                // open after 5 failures...
		FailurePeriod:    10 * time.Second, // ...with no gap longer than 10s between them
		OpenTimeout:      30 * time.Second, // stay open for 30s, then probe
		SuccessThreshold: 3,                // close again after 3 successful probes
	})
	if err != nil {
		log.Fatal(err)
	}

	err = cb.Do(ctx, "cb:example-api", func() error {
		resp, err := http.Get("https://example.com/health")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 500 {
			return fmt.Errorf("example api: %s", resp.Status)
		}
		return nil
	})
	switch {
	case errors.Is(err, circuitbreaker.ErrOpened):
		fmt.Println("breaker open: skipped the call, serve a fallback")
	case err != nil:
		fmt.Println("call failed:", err)
	default:
		fmt.Println("ok")
	}
}
```

To keep a function's signature, wrap it with `circuitbreaker.Func`:

```go
getUser = circuitbreaker.Func(getUser, cb, func(ctx context.Context, id string) (string, error) {
	return "cb:users-api", nil // which breaker to use for this request
})
```

More runnable examples are in [`example_test.go`](example_test.go) and on [pkg.go.dev](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/circuitbreaker).

## How it works

```
          failures >= FailureThreshold
 Closed ──────────────────────────────▶ Opened
   ▲                                     │  ▲
   │ successes >= SuccessThreshold       │  │ any failure
   │                    OpenTimeout over ▼  │
   └───────────────────────────────── HalfOpen
```

| State | `Do` runs `fn`? | What is recorded |
|---|---|---|
| `Closed` | yes | Failures are added to a counter. Successes cost nothing extra |
| `Opened` | no, returns `ErrOpened` | Nothing. After `OpenTimeout`, the next `Do` moves the breaker to `HalfOpen` |
| `HalfOpen` | yes (probe) | One failure reopens the breaker. `SuccessThreshold` successes close it |
| `Disabled` | yes | Nothing. Only set manually with `SetStatus` |
| `ForcedOpen` | no, returns `ErrOpened` | Nothing. Only set manually, and never times out |

Each breaker key is one Redis hash with these fields:

- `status`: the current state.
- `counter`: failures while Closed, or successes while HalfOpen. The field expires after `FailurePeriod` or `SuccessPeriod` without updates.
- `timeout`: when an Opened breaker may start probing.
- `generation`: incremented on every state change.

A call does the following:

1. `cb_begin` reads the state, moving Opened to HalfOpen once the timeout is over, and returns the state and generation.
2. `fn` runs in your process.
3. `cb_commit` records the result, but only after a failure or a HalfOpen probe. A result whose generation is older than the current one is ignored, so a slow call that started before a reset can't reopen the breaker.

All timing uses the Redis server clock.

**Failure weight.** A failed call adds `1 + FailureCount(err) + SlowCallCount(duration)` to the counter. With the defaults, a `context.DeadlineExceeded` counts as 3, and slow failures count more. Successful calls are never counted as failures, however slow they are.

## Configuration

`circuitbreaker.Config` is passed by value. Zero fields take their defaults.

| Field | Default | Meaning |
|---|---|---|
| `FailureThreshold` | `100` | Failure weight at which Closed changes to Opened |
| `FailurePeriod` | `1s` | Failure count is forgotten after this long without a new failure (at least 1ms) |
| `SuccessThreshold` | `20` | Successful probes at which HalfOpen changes to Closed |
| `SuccessPeriod` | `1s` | Success count is forgotten after this long without a new success (at least 1ms) |
| `OpenTimeout` | `1m` | How long to stay Opened before probing (at least 1ms) |
| `FailureCount` | `+2` for `context.DeadlineExceeded` | Extra weight for a failure. Return `0` to count every error once |
| `SlowCallCount` | `+1` over 1s, `+2` from 30s, `+4` from 1m | Extra weight for a slow *failed* call |

The defaults suit high-traffic dependencies. For low traffic, lower `FailureThreshold` and raise `FailurePeriod`. The hooks must be safe for concurrent use and must not return negative numbers.

## Errors

| Error | Meaning | What to do |
|---|---|---|
| `ErrOpened` | The breaker is Opened or ForcedOpen, and `fn` was **not** called | Fail fast, or serve a cached or default response |
| The error from `fn` | `fn` ran and failed | Handle it as usual. It was counted |
| Error mentioning `circuitbreaker.Setup` | The Lua functions are not loaded on that Redis | Call `Setup` at startup |
| Other Redis errors | Redis is unreachable | See below |

Use `errors.Is`. After a failed call, a Redis error from recording the result is joined with `fn`'s error.

**If Redis is down,** `Do` returns the Redis error **without calling `fn`**, which means it fails closed. If you prefer to fail open, call the dependency directly when `Do` fails before running `fn`. You can track whether `fn` ran with a flag set inside the closure.

## Manual control

```go
cb.SetStatus(ctx, "cb:payments-api", circuitbreaker.ForcedOpen) // stop all calls during an incident
cb.SetStatus(ctx, "cb:payments-api", circuitbreaker.Closed)     // reset
cb.SetStatus(ctx, "cb:payments-api", circuitbreaker.Disabled)   // bypass the breaker
status, err := cb.Status(ctx, "cb:payments-api")
```

`Status` returns the stored state. An Opened breaker whose timeout has passed still reads `Opened` until the next `Do`.

## Deploying and upgrading

- Call `Setup(ctx, client)` on every application start. It runs `FUNCTION LOAD REPLACE` for the `circuitbreaker` library, so it is idempotent and upgrades the functions after you bump this module.
- Function names, arguments and the hash layout stay backward compatible, so old and new instances can run side by side during a rolling deploy.
- **Redis Cluster:** pass a `*redis.ClusterClient` to `Setup` and `New`. `Setup` loads the library on every master. Every operation touches exactly one key, so any key works. A `*redis.Ring` is supported the same way.

## Guarantees and caveats

- **HalfOpen does not limit concurrency.** Every call made during HalfOpen runs as a probe. If the dependency can only take a trickle of traffic, add your own limiter, for example the `ratelimit` module.
- Every non-nil error from `fn` counts. Don't return errors that aren't the dependency's fault, such as validation errors or 404s.
- `Do` makes one Redis round trip before every call, and a second one after failures and probes.
- A failed result is recorded with its own 5-second timeout, detached from the caller's context, so cancelled requests are still counted.
- If recording a *successful* probe fails, `Do` still returns `nil`, so a call that succeeded is never reported as failed.
- Keys never expire. Use a small, fixed set of keys (one per dependency, not one per request). Deleting a key resets the breaker.
- If `fn` panics, nothing is recorded.
- State lives on one Redis primary. A failover that loses writes can reset a breaker.

## Running the tests

The tests start a Redis container with Docker (via `dockertest`):

```sh
go test -race ./...
```
