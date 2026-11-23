# throttle

Bound how many operations run at once, optionally queueing a limited backlog
of callers that wait for a free slot.

**Use it for:** protecting a connection pool, a CPU-heavy task or a downstream
API from too many simultaneous calls.

**Don't use it for:** limiting how often something happens over time
([`ratelimit`](../ratelimit) limits rate, not concurrency), or failing fast
after repeated errors ([`circuitbreaker`](../circuitbreaker); its half-open
state can also be limited with `HalfOpenMaxCalls`). Limits are per process.

## Installation

```sh
go get github.com/alextanhongpin/core/sync/throttle
```

[API reference](https://pkg.go.dev/github.com/alextanhongpin/core/sync/throttle)

## Quick start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/throttle"
)

func main() {
	t, err := throttle.New(throttle.Config{
		Limit: 2, BacklogLimit: 1, BacklogTimeout: time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := t.Do(context.Background(), func(ctx context.Context) error {
				time.Sleep(50 * time.Millisecond) // your operation; honor ctx
				return nil
			})
			switch {
			case errors.Is(err, throttle.ErrCapacityExceeded):
				fmt.Println("rejected: running and backlog slots are full")
			case err != nil:
				fmt.Println("error:", err)
			}
		}()
	}
	wg.Wait()
	fmt.Println("done")
}
```

Two of the five calls are admitted immediately, one waits in the backlog, and
two are rejected, so the program prints `rejected: running and backlog slots are full`
twice and then `done`.

## API

```go
type Config struct {
	Limit          int           // max concurrent calls (default 1000)
	BacklogLimit   int           // extra callers allowed to wait; 0 = no queue
	BacklogTimeout time.Duration // max wait for a slot; 0 = do not wait
}
func DefaultConfig() Config // Limit 1000, BacklogLimit 100, BacklogTimeout 10s
func (Config) WithDefaults() Config
func (Config) Validate() error

func New(cfg Config) (*Throttler, error)
func MustNew(cfg Config) *Throttler // panics on invalid cfg
func (*Throttler) Do(ctx context.Context, fn func(context.Context) error) error

func Func[K, V any](fn func(ctx context.Context, req K) (V, error), t interface {
	Do(context.Context, func(context.Context) error) error
}) func(ctx context.Context, req K) (V, error)

var ErrCapacityExceeded, ErrTimeout error
```

## Behavior and guarantees

- **Admission.** `Do` rejects with `ErrCapacityExceeded` when `Limit +
  BacklogLimit` calls are already running or waiting. Otherwise it runs `fn`
  synchronously once a slot is free, or returns `ErrTimeout` if
  `BacklogTimeout` expires first. With the zero `Config`, `BacklogTimeout` is 0
  and a busy throttler returns `ErrTimeout` immediately; use `DefaultConfig`
  for a queued setup.
- **Cancellation.** An already-canceled `ctx` returns its cause without
  running `fn`. `ctx` ending while waiting returns its cause. `fn` receives the
  original caller context (the timeout covers waiting only, not execution), and
  must honor it itself.
- **Ordering.** Waiters are not served in FIFO order.
- **Concurrency.** Safe for concurrent use; `fn` calls run concurrently up to
  `Limit`. Slots are released when `fn` returns or panics.
- **Errors.** `fn`'s error is returned as is; use `errors.Is` for
  `ErrCapacityExceeded`, `ErrTimeout`, `context.Canceled` and
  `context.DeadlineExceeded`.
- **`Func`.** With retry outside the throttle, one permit covers attempts and
  backoff; inside retry, each attempt takes its own permit.
- **Limitations.** Limits are per `Throttler` instance, not shared across
  processes. No reentrancy: calling `Do` from inside `fn` on a full throttler
  can deadlock or time out.

## Testing

```sh
go vet ./... && go test -race -count=1 ./...
```

Some tests use the private `evaltest` module and golden files in `testdata/`.
