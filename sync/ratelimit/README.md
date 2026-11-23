# ratelimit

Per-key admission control in one process: how many requests a key may make
per period. Offers GCRA (smooth) and fixed-window limiters, plus function and
HTTP middleware adapters.

**Use it for:** per-user or per-IP API quotas on a single service instance.

**Don't use it for:** bounding how many calls run at the same time
([`throttle`](../throttle) limits concurrency, not rate), tripping on a failing
dependency ([`circuitbreaker`](../circuitbreaker)), or quotas shared by several
processes (use [`dsync/ratelimit`](../../dsync/ratelimit), the Redis-shared
variant).

## Installation

```sh
go get github.com/alextanhongpin/core/sync/ratelimit
```

[API reference](https://pkg.go.dev/github.com/alextanhongpin/core/sync/ratelimit)

## Quick start

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

func main() {
	limiter, err := ratelimit.NewGCRA(ratelimit.Config{
		Limit: 5, Period: time.Minute, Burst: 2, // 5 per minute, 2 extra in a burst
	})
	if err != nil {
		log.Fatal(err)
	}
	for i := 1; i <= 5; i++ {
		res := limiter.Limit("user:123")
		if !res.Allow {
			fmt.Printf("request %d: denied, retry after %s\n", i, res.RetryAfter.Round(time.Second))
			continue
		}
		fmt.Printf("request %d: allowed, %d remaining\n", i, res.Remaining)
	}
}
```

Output:

```
request 1: allowed, 2 remaining
request 2: allowed, 1 remaining
request 3: allowed, 0 remaining
request 4: denied, retry after 12s
request 5: denied, retry after 12s
```

## API

```go
type Config struct {
	Limit  int           // admissions per Period (default 100)
	Period time.Duration // default 1m
	Burst  int           // extra instantaneous capacity (GCRA only); default 0
}
func DefaultConfig() Config
func (Config) WithDefaults() Config
func (Config) Validate() error

func NewGCRA(cfg Config) (*GCRA, error)
func MustNewGCRA(cfg Config) *GCRA
func NewFixedWindow(cfg Config) (*FixedWindow, error)
func MustNewFixedWindow(cfg Config) *FixedWindow

// Both limiters:
func (l) Allow(key string) bool
func (l) AllowN(key string, n int) bool
func (l) Limit(key string) *Result
func (l) LimitN(key string, n int) *Result // consumes n tokens if they all fit
func (l) Clear()    // delete expired keys
func (l) Size() int // number of tracked keys

type Result struct { Allow bool; Remaining int; ResetAfter, RetryAfter time.Duration; Limit int }

func Func[K, V any](fn func(context.Context, K) (V, error), cfg FuncConfig[K]) func(context.Context, K) (V, error) // ErrTooManyRequests
func HTTP(next http.Handler, cfg FuncConfig[*http.Request]) http.Handler // 429 + Retry-After
type FuncConfig[T any] struct { RateLimiter; KeyFn func(context.Context, T) (string, error) }
var ErrTooManyRequests error
```

## Behavior and guarantees

- **Concurrency.** Both limiters are safe for concurrent use. `KeyFn` and
  downstream functions are caller-owned and may run concurrently.
- **Time.** Elapsed time is measured from a base instant captured at
  construction using Go's monotonic clock, so wall-clock adjustments do not
  affect admission.
- **Keys and quantities.** Empty keys and negative quantities are denied.
  `n == 0` inspects the allowance without consuming tokens and does not create
  state for an unseen key. A batch is admitted whole or not at all; a rejected
  batch consumes nothing.
- **FixedWindow.** A window starts at a key's first admission attempt after the
  previous one expired and lasts `Period`. `Burst` is ignored. Up to twice
  `Limit` can be admitted across a window boundary.
- **GCRA.** Emissions are spaced `Period/Limit` apart; up to `Burst+1` can be
  admitted at once. A batch larger than `Burst+1` can never be admitted.
  `ResetAfter` currently equals `RetryAfter`, not the time to fully replenish
  capacity. Construction fails for a sub-nanosecond interval or an overflowing
  burst.
- **Errors.** `Allow` has no error; denial is `false`. `Func` returns
  `ErrTooManyRequests`. `HTTP` replies 429 with `Retry-After` rounded up to
  whole seconds, and sets `X-RateLimit-*` headers on admitted requests.
- **Memory.** Per-key state is kept until `Clear` removes expired entries; no
  background goroutine runs. Call `Clear` periodically if keys are unbounded.
- **Limitations.** Quotas are per instance, not shared across processes.
  With retry outside `Func`, every attempt consumes quota.

## Testing

```sh
go vet ./... && go test -race -count=1 ./...
```

Some tests use the private `evaltest` module and golden files in `testdata/`.
