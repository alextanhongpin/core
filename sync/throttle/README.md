# throttle

A small Go package for rate-limiting work with a primary concurrency limit and an optional backlog queue. It is implemented with buffered channels and provides back-pressure via `ErrCapacityExceeded` and timeout via `ErrTimeout`.

## Overview

`Throttler` controls how many operations can run concurrently.

* `Limit` – number of concurrent slots in the primary pool.
* `BacklogLimit` – additional queued slots. When the primary pool is exhausted a caller can still take a backlog token and then wait for a primary slot to free.
* `BacklogTimeout` – maximum time a caller is allowed to wait for a primary slot after acquiring a backlog token. The wait is enforced with `context.WithTimeoutCause` and fails with `ErrTimeout`.

The design is deliberately simple:
* tokens are `struct{}` values in two buffered channels
* `New` pre-fills `ch` with `Limit` tokens and `backlogCh` with `Limit + BacklogLimit` tokens
* `Do` tries to take a backlog token non-blocking, then takes a primary token with timeout, runs `fn`, and returns both tokens on exit

This gives you a bounded concurrency primitive with graceful degradation: when both pools are empty `Do` returns `ErrCapacityExceeded` immediately.

## Installation

```bash
go get github.com/alextanhongpin/core/sync/throttle
```

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"time"
	"github.com/alextanhongpin/core/sync/throttle"
)

func main() {
	cfg := throttle.DefaultConfig()
	cfg.Limit = 3
	cfg.BacklogLimit = 2
	cfg.BacklogTimeout = 5 * time.Second

	t := throttle.New(cfg)

	err := t.Do(context.Background(), func(ctx context.Context) error {
		fmt.Println("doing work")
		time.Sleep(1 * time.Second)
		return nil
	})

	if err != nil {
		fmt.Println("throttled:", err)
	}
}
```

## API

### Config

```go
type Config struct {
	BacklogLimit   int
	BacklogTimeout time.Duration
	Limit          int
}
```

`DefaultConfig()` returns a sensible default:

```go
Limit:           1000
BacklogLimit:    100
BacklogTimeout:  10 * time.Second
```

`Validate()` checks:
* `Limit > 0`
* `BacklogLimit >= 0`
* `BacklogTimeout >= 0`

### New

```go
func New(cfg *Config) *Throttler
```

Creates a new throttler. `nil` config is replaced with `DefaultConfig()`. Panics if `Limit <= 0`.

### Do

```go
func (t *Throttler) Do(ctx context.Context, fn func(context.Context) error) error
```

Attempts to acquire a token within `t.BacklogTimeout`.

* Returns `nil` on success, `fn` is executed.
* Returns `ErrTimeout` if the context is cancelled or the configured timeout expires.
* Returns `ErrCapacityExceeded` if `backlogCh` is empty at call time.

Acquisition order:
1. non-blocking take from `backlogCh`; if empty return `ErrCapacityExceeded`
2. take from `ch` with timeout derived from `BacklogTimeout` and the incoming context
3. run `fn`
4. return tokens to both channels on exit via deferred `select` with default to avoid deadlock

### Errors

```go
var (
	ErrTimeout          = errors.New("throttle: timeout")
	ErrCapacityExceeded = errors.New("throttle: capacity exceeded")
)
```

## Testing

The package ships with evaltest data-driven tests:

```bash
go test -run TestThrottleDo
```

See `README_TESTS.md` for details.

## Example scenarios

* **Burst protection**: `Limit=10, BacklogLimit=0` – rejects immediately once 10 goroutines are running.
* **Graceful queue**: `Limit=5, BacklogLimit=20, BacklogTimeout=30s` – queues up to 20 extra requests for up to 30 seconds.
* **Hard reject**: `Limit=3, BacklogLimit=0` – no queuing, immediate rejection.

## Notes

* The throttler has no `Close` method; channels are left for GC when the throttler is discarded.
* Tokens are returned via `select` with default to avoid dead-lock on shutdown.
* For production use consider adding metrics around `ErrCapacityExceeded` and `ErrTimeout` rates.

## Func Helper

The package provides a generic helper in `func.go` for wrapping request functions with throttling:

```go
type fun[K, V any] = func(ctx context.Context, req K) (V, error)

type throttler interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

func Func[K, V any](fn fun[K, V], t throttler) fun[K, V]
```

`Func` adapts `func(context.Context, K) (V, error)` to run under a `throttler`. The returned function acquires a throttle token via `t.Do` before invoking the original `fn`. This is useful for decorating service methods with rate limiting without changing their signatures.
