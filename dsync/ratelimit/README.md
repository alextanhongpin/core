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

Requirements: **Go 1.25.6+** (from `go.mod`) and **Redis 7.0+** (Redis Functions).

```sh
go get github.com/alextanhongpin/core/dsync/ratelimit
```

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

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // your Redis address
	})
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

Expected output: requests 1 to 3 are allowed (burst of `Burst+1`), with `2`, `1`, `0` left. Requests 4 to 10 run within a few milliseconds, so they are denied with a retry of about one interval (`Period/Limit` = 200ms), printed as `retry after 200ms` or `199ms`. The state lives in Redis and expires once the key is idle (here about 600ms after the last allowed request): a second run a second later prints the same, but a run started right away is denied from request 1 with a shorter retry such as `153ms`.

### HTTP middleware

```go
// RateLimit answers 429 Too Many Requests with a Retry-After header.
func RateLimit(limiter ratelimit.RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, err := limiter.Limit(r.Context(), "rl:api:"+r.Header.Get("X-API-Key"))
		if err != nil {
			// Redis is unavailable. This fails closed; you may prefer to fail open.
			http.Error(w, "rate limiter unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		if !res.Allow {
			// Retry-After takes whole seconds. Round up so clients don't retry early.
			secs := int(math.Ceil(res.RetryAfter.Seconds()))
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

The package does no rounding: `RetryAfter` is a `time.Duration` in whole milliseconds (for example `199ms`). Converting it to seconds is up to you; rounding up, as above, turns `199ms` into `Retry-After: 1`. A runnable version is `Example_httpMiddleware` in `example_test.go`.

## API

```go
func Setup(ctx context.Context, client redis.UniversalClient) error
const LibraryName = "ratelimit" // Redis function library installed by Setup

func NewFixedWindow(client redis.UniversalClient, cfg Config) (*FixedWindow, error)
func MustNewFixedWindow(client redis.UniversalClient, cfg Config) *FixedWindow // panics on error
func NewGCRA(client redis.UniversalClient, cfg Config) (*GCRA, error)
func MustNewGCRA(client redis.UniversalClient, cfg Config) *GCRA // panics on error

// Implemented by *FixedWindow and *GCRA.
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	AllowN(ctx context.Context, key string, n int) (bool, error) // all or nothing
	Limit(ctx context.Context, key string) (*Result, error)
	LimitN(ctx context.Context, key string, n int) (*Result, error)
}

type Result struct {
	Allow      bool          // the requests were admitted and consumed
	Remaining  int           // how many more requests fit right now
	ResetAfter time.Duration // until the key is back to full capacity (FixedWindow: until the window ends)
	RetryAfter time.Duration // wait before the same request can succeed; 0 if allowed
}

type Config struct {
	Limit  int
	Period time.Duration
	Burst  int
}
func DefaultConfig() Config                // Config{Limit: 100, Period: time.Minute}
func (c Config) WithDefaults() Config      // zero Limit/Period replaced by the defaults
func (c Config) Validate() error           // does not apply defaults

var ErrNegative, ErrEmptyKey, ErrNotSetup error
```

`LimitN(ctx, key, 0)` inspects the current state without consuming anything. `Allow`/`AllowN` return `Result.Allow`. All `Result` durations have millisecond precision.

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
| `Period` | `1m` | Window length. Must be at least 1ms; truncated to whole milliseconds (`1500*time.Microsecond` behaves as `1ms`, `999*time.Microsecond` is rejected) |
| `Burst` | `0` | GCRA only: extra requests allowed at once. FixedWindow ignores it |

Validation happens in `NewFixedWindow` and `NewGCRA`, which return an error. `MustNewFixedWindow` and `MustNewGCRA` panic instead. GCRA also requires `Period/Limit >= 1ms`: 1000 per second is accepted, 2000 per second is not.

**GCRA does not admit `Limit` requests at once.** With `Burst: 0` it admits one request immediately, then one every `Period/Limit`. So the default `Config{}` (100 per 1m) allows 1 request, then one every 600ms, and the first `Result` already shows `Remaining: 0`. Choose `Burst` as "how many requests may arrive at once, minus one":

| You want | Config |
|---|---|
| Strict spacing, no bursts | `Limit: 10, Period: time.Second` (one per 100ms) |
| Up to 10 at once, then 10/s sustained | `Limit: 10, Period: time.Second, Burst: 9` |
| "100 per minute" that may all come at once | `Limit: 100, Period: time.Minute, Burst: 99`, or use `FixedWindow` |

## Errors

A denial is **not** an error. `Allow` returns `false, nil` and `Limit` returns `Result{Allow: false}`.

| Error | Cause | What to do |
|---|---|---|
| `ErrNegative` | `n < 0` | Fix the caller |
| `ErrEmptyKey` | `key == ""` | Fix the caller |
| `ErrNotSetup` | The Lua functions are not loaded on that Redis. Message: `ratelimit: Redis function missing, call ratelimit.Setup first: ERR Function not found`. Check with `errors.Is(err, ratelimit.ErrNotSetup)` | Call `Setup` at startup |
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
