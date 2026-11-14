# circuitbreaker

A circuit breaker whose state is stored in Redis. When a dependency starts failing, **every** instance of your service stops calling it at the same time, and they all recover together.

**Use it when** several replicas call the same dependency (a payments API, a database, another microservice) and should back off together instead of each one finding out on its own.
**Don't use it when** you run a single process (an in-memory breaker is faster and needs no Redis), or when you need a hard limit on concurrent probes. Half-open does not limit concurrency, see [caveats](#guarantees-and-caveats).

## Install

```sh
go get github.com/alextanhongpin/core/dsync/circuitbreaker
```

Requires **Go 1.26+** and **Redis 7.4+** (it uses Redis Functions and hash-field expiration, `HPEXPIRE`).

## Quick start

The dependency here is a local function that always fails, so you can see the breaker open.

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"}) // your Redis address
	defer client.Close()

	// Install the Lua functions. Safe to call on every start.
	if err := circuitbreaker.Setup(ctx, client); err != nil {
		log.Fatal(err)
	}

	cb, err := circuitbreaker.New(client, circuitbreaker.Config{
		FailureThreshold: 3,                // open after 3 failures...
		FailurePeriod:    10 * time.Second, // ...with no gap longer than 10s between them
		OpenTimeout:      30 * time.Second, // stay open for 30s, then probe
		SuccessThreshold: 2,                // close again after 2 successful probes
	})
	if err != nil {
		log.Fatal(err)
	}

	// Stand-in for a real outbound call (HTTP, gRPC, database...).
	callPayments := func() error { return errors.New("payments api unavailable") }

	for i := 1; i <= 4; i++ {
		err := cb.Do(ctx, "cb:quickstart", callPayments)
		switch {
		case errors.Is(err, circuitbreaker.ErrOpened):
			fmt.Printf("call %d: rejected, serving fallback\n", i)
		case err != nil:
			fmt.Printf("call %d: failed: %v\n", i, err)
		default:
			fmt.Printf("call %d: ok\n", i)
		}
	}

	status, err := cb.Status(ctx, "cb:quickstart")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("status:", status)
}
```

Expected output on the first run:

```
call 1: failed: payments api unavailable
call 2: failed: payments api unavailable
call 3: failed: payments api unavailable
call 4: rejected, serving fallback
status: opened
```

The state lives in Redis, so a **second run within 30s** prints `rejected` for all four calls. After 30s the first call runs as a HalfOpen probe, fails, and reopens the breaker, so you get one `failed` line then three `rejected`. Run `redis-cli DEL cb:quickstart` to start fresh.

To keep a function's signature, wrap it with `circuitbreaker.Func`:

```go
getUser = circuitbreaker.Func(getUser, cb, func(ctx context.Context, id string) (string, error) {
	return "cb:users-api", nil // which breaker to use for this request
})
```

More runnable examples, including one using `httptest`, are in [`example_test.go`](example_test.go).

## API

```go
func Setup(ctx context.Context, client redis.UniversalClient) error
const LibraryName = "circuitbreaker" // Redis function library loaded by Setup

func New(client redis.UniversalClient, cfg Config) (*CircuitBreaker, error) // error if cfg is invalid or client is nil
func MustNew(client redis.UniversalClient, cfg Config) *CircuitBreaker      // panics instead
func DefaultConfig() Config
func (Config) WithDefaults() Config // fills zero fields from DefaultConfig
func (Config) Validate() error      // does not apply defaults

func (*CircuitBreaker) Do(ctx context.Context, key string, fn func() error) error
func (*CircuitBreaker) SetStatus(ctx context.Context, key string, status Status) error
func (*CircuitBreaker) Status(ctx context.Context, key string) (Status, error) // missing key = Closed

// Func wraps fn so every call goes through cb.Do; keyFn picks the breaker key.
func Func[K, V any](
	fn func(ctx context.Context, req K) (V, error),
	cb interface{ Do(context.Context, string, func() error) error },
	keyFn func(context.Context, K) (string, error),
) func(ctx context.Context, req K) (V, error)

type Status int // Unknown, Closed, HalfOpen, Opened, Disabled, ForcedOpen
func (Status) String() string // "closed", "half-open", "opened", "disabled", "forced-open"
func (Status) Int() int

var ErrOpened   = errors.New("cb: opened")
var ErrNotSetup = errors.New("circuitbreaker: Redis function missing, call circuitbreaker.Setup first")
```

`Do` returns `fn`'s error (use `errors.Is`), `ErrOpened` if `fn` was skipped, the context's error if `ctx` is already done, or a Redis error. Details in [Errors](#errors).

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
| `FailureCount` | `+2` for `context.DeadlineExceeded`, else `0` | `func(cause error) int`. Extra weight for a failed call |
| `SlowCallCount` | see below | `func(duration time.Duration) int`. Extra weight for a slow *failed* call |

The defaults suit high-traffic dependencies. For low traffic, lower `FailureThreshold` and raise `FailurePeriod`. The hooks must be safe for concurrent use and must not return negative numbers (a negative total weight makes `Do` return an error). `duration` is how long `fn` took. The default `SlowCallCount` is:

```go
func(d time.Duration) int {
	switch {
	case d >= time.Minute:
		return 4
	case d >= 30*time.Second:
		return 2
	case d > time.Second: // strictly more than 1s
		return 1
	}
	return 0
}
```

To count every error exactly once, set both hooks to return 0:

```go
cfg := circuitbreaker.Config{
	FailureThreshold: 5,
	FailureCount:     func(error) int { return 0 },
	SlowCallCount:    func(time.Duration) int { return 0 },
}
```

## Errors

| Error | Meaning | What to do |
|---|---|---|
| `ErrOpened` | The breaker is Opened or ForcedOpen, and `fn` was **not** called | Fail fast, or serve a cached or default response |
| The error from `fn` | `fn` ran and failed | Handle it as usual. It was counted |
| `ErrNotSetup` (wrapped) | The Lua functions are not loaded on that Redis (Setup never ran, or the server was flushed). Message starts with `circuitbreaker: Redis function missing, call circuitbreaker.Setup first:` | Call `Setup` at startup |
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
