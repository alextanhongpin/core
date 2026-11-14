# ratelimit

Distributed rate limiting backed by Redis. Every instance of your service shares the same quota for a given key, such as a user, an API key or an IP address.

**Use it when** several replicas must enforce one quota together, for example "100 requests per minute per API key".
**Don't use it when** a single process is enough (use `golang.org/x/time/rate`, which needs no network call), or when you need strict durability. If Redis loses data, the counters reset.

## Choosing an algorithm

| | `FixedWindow` | `GCRA` |
|---|---|---|
| Rule | At most `Limit` requests per `Period` window | One request every `Period/Limit` on average, plus `Burst` extra at once |
| Window | Starts at a key's first request, then resets | No windows; capacity refills continuously |
| Bursts | Up to `2×Limit` across a window boundary | At most `Burst+1` at once, about `Limit+Burst` per `Period` |
| Redis state | One counter per key | One timestamp per key |
| Pick it for | Simple quotas ("1000 per day") | Smooth traffic shaping ("10 per second") |

If you are unsure, use **GCRA**.

## Install

```sh
go get github.com/alextanhongpin/core/dsync/ratelimit
```

Requires Redis 7.0 or newer (Redis Functions).

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()

	// Install the Lua functions. This is safe to call on every start.
	if err := ratelimit.Setup(ctx, client); err != nil {
		log.Fatal(err)
	}

	// 5 requests per second, and up to 2 extra in a burst.
	limiter, err := ratelimit.NewGCRA(client, ratelimit.Config{
		Limit:  5,
		Period: time.Second,
		Burst:  2,
	})
	if err != nil {
		log.Fatal(err)
	}

	for i := 1; i <= 10; i++ {
		res, err := limiter.Limit(ctx, "user:123")
		if err != nil {
			log.Fatal(err) // Redis failure, not a denial.
		}
		if !res.Allow {
			fmt.Printf("request %d denied, retry after %s\n", i, res.RetryAfter)
			continue
		}
		fmt.Printf("request %d allowed, %d left\n", i, res.Remaining)
	}
}
```

For an HTTP middleware that returns `429 Too Many Requests` with a `Retry-After` header, see the [`httpMiddleware` example](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/ratelimit#example-package-HttpMiddleware) in `example_test.go`.

## API at a glance

| Call | Returns | Use for |
|---|---|---|
| `Allow(ctx, key)` | `bool` | Checking a single request |
| `AllowN(ctx, key, n)` | `bool` | Checking `n` units at once, all or nothing |
| `Limit(ctx, key)` / `LimitN(ctx, key, n)` | `*Result` | Getting `Remaining`, `RetryAfter` and `ResetAfter` for headers |

Calling `LimitN(ctx, key, 0)` inspects the current state without consuming anything. Both limiters implement the `RateLimiter` interface.

`Result` fields:

| Field | Meaning |
|---|---|
| `Allow` | The requests were admitted and consumed |
| `Remaining` | How many more requests fit right now |
| `RetryAfter` | How long to wait before the same request can succeed (0 if allowed) |
| `ResetAfter` | How long until the key is back to full capacity (FixedWindow: until the window ends) |

## How it works

Each call runs one Lua function inside Redis (`rl_fixed_window` or `rl_gcra`). It reads the key, decides, updates it and returns the result in a single atomic step. Concurrent callers can't double-spend capacity. Time comes from Redis's `TIME` command, so clock differences between your servers don't matter.

**Fixed window.** The key holds a counter. The first admitted request creates it with a TTL of `Period`. Later requests increment it while `count + n <= Limit`. When the TTL expires the counter disappears and a new window starts.

```
Limit=3  |■■■ denied denied|■■■ ...      (window = Period, starts at 1st request)
```

**GCRA (Generic Cell Rate Algorithm).** Each request "costs" `interval = Period/Limit` of time. The key stores a *theoretical arrival time* (TAT), which is when the key would be fully idle again. A request is allowed if `TAT - now <= (Burst+1-n) × interval`. When it is admitted, TAT moves forward by `n × interval`. The key expires when TAT is reached, so idle keys cost nothing.

```
Limit=5/s, Burst=2: interval=200ms, up to 3 instantly, then 1 every 200ms
```

A denied request never consumes capacity, for either algorithm.

## Configuration

`ratelimit.Config` is passed by value. Zero fields take their defaults.

| Field | Default | Meaning |
|---|---|---|
| `Limit` | `100` | Requests allowed per `Period` |
| `Period` | `1m` | Window length (must be at least 1ms, millisecond precision) |
| `Burst` | `0` | GCRA only: extra requests allowed at once. FixedWindow ignores it |

Validation happens in `NewFixedWindow` and `NewGCRA`, which return an error. `MustNew...` panics instead. GCRA also requires `Period/Limit >= 1ms`: 1000 per second is accepted, 2000 per second is not.

## Errors

A denial is **not** an error. `Allow` returns `false, nil` and `Limit` returns `Result{Allow: false}`.

| Error | Cause | What to do |
|---|---|---|
| `ErrNegative` | `n < 0` | Fix the caller |
| `ErrEmptyKey` | `key == ""` | Fix the caller |
| Error mentioning `ratelimit.Setup` | The Lua functions are not loaded on that Redis | Call `Setup` at startup |
| Any other error | Redis is unreachable or a context was cancelled | Decide whether to fail open (allow) or fail closed (reject or return 503) |

## Deploying and upgrading

- Call `Setup(ctx, client)` on every application start. It runs `FUNCTION LOAD REPLACE` for the `ratelimit` library, so it is idempotent and upgrades the functions after you bump this module.
- Function names and arguments stay backward compatible, so old and new instances can run side by side during a rolling deploy.
- Redis persists and replicates functions the same way it does data. A fresh Redis without persistence starts with no functions, which is one more reason to call `Setup` on start.
- **Redis Cluster:** pass a `*redis.ClusterClient` to `Setup` and the constructors. `Setup` loads the library on every master. Each call touches exactly one key, so any key works in Cluster. A `*redis.Ring` is supported the same way.

## Guarantees and caveats

- Decisions are atomic for each key on one Redis primary. If a failover loses recent writes, a few extra requests can get through.
- Keys are used as given. Add your own prefix (for example `"rl:login:" + userID`) so they don't collide with other data.
- `AllowN` is all-or-nothing. A batch larger than the capacity (`Limit` for FixedWindow, `Burst+1` for GCRA) is never allowed. GCRA reports `RetryAfter == 0` for such a batch, and FixedWindow reports the end of the current window.
- Every call is a network round trip to Redis. Set timeouts on the context or the client.
- No background goroutines are started, and the client is never closed for you.

## Running the tests

The tests start a Redis container with Docker (via `dockertest`):

```sh
go test -race ./...
```
