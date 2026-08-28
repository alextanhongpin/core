# retry

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/sync/retry.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/sync/retry)
[![Go Report Card](https://goreportcard.com/badge/github.com/alextanhongpin/core/sync/retry)](https://goreportcard.com/report/github.com/alextanhongpin/core/sync/retry)

A Go retry package with configurable backoff, context-aware cancellation, adaptive token-bucket throttling and an `http.RoundTripper` decorator.

## Features

- **Retry with `Retry.Do` and generic `Func` wrapper**  
  `Retry.Do` retries a `func(ctx) error`. `Func` wraps `func(ctx, req) (V, error)` into a retryable function.
- **Backoff strategies**  
  `ConstantBackoff`, `LinearBackoff`, `ExponentialBackoff` with full jitter. Implement `Backoff` to provide a custom strategy.
- **Adaptive throttling**  
  Token bucket `Throttler` / `Limiter` to avoid retry storms. A noop limiter is available.
- **Error classification**  
  `Retryable` predicate, `NonRetryableErrors` helper for permanent errors. Sentinel errors: `ErrLimitExceeded`, `ErrThrottled`, `ErrCanceled`.
- **HTTP integration**  
  `NewRoundTripper` wraps an `http.RoundTripper` and retries on error or retryable status codes. Response bodies are closed on retry and requests with `GetBody` are safely replayed.

## Installation

```bash
go get github.com/alextanhongpin/core/sync/retry
```

## Usage

### Basic retry

```go
package main

import (
    "context"
    "log"
    "time"
    "github.com/alextanhongpin/core/sync/retry"
)

func main() {
    ctx := context.Background()

    cfg := retry.DefaultConfig()
    cfg.Attempts = 3
    cfg.Backoff = retry.NewExponentialBackoff(100*time.Millisecond, 2*time.Second)

    r := retry.New(cfg)

    err := r.Do(ctx, func(ctx context.Context) error {
        return performOperation(ctx)
    })
    if err != nil {
        log.Printf("operation failed: %v", err)
    }
}
```

### Generic function wrapper

```go
type Req struct{ ID string }
type Res struct{ Name string }

fn := func(ctx context.Context, r Req) (Res, error) { ... }

r := retry.New(retry.DefaultConfig())
wrapped := retry.Func(fn, r)

res, err := wrapped(ctx, Req{ID: "1"})
```

### Backoff

```go
retry.NewConstantBackoff(500*time.Millisecond)
retry.NewLinearBackoff(100*time.Millisecond)      // At(n) = period * n
retry.NewExponentialBackoff(100*time.Millisecond, 30*time.Second) // full jitter
```

`Backoff` interface:
```go
type Backoff interface {
    At(attempts int) time.Duration
}
```

### Throttling

```go
throttler := retry.NewThrottler(&retry.ThrottlerConfig{
    MaxTokens:   10,
    TokenRatio:  0.1,
})
cfg := retry.DefaultConfig()
cfg.Throttler = throttler
```

`NewNoopThrottler` disables throttling.

### Error classification

```go
cfg.Retryable = retry.NonRetryableErrors(context.Canceled, context.DeadlineExceeded)

// custom predicate
cfg.Retryable = func(err error) (error, bool) {
    // return wrapped error and retry=false for permanent errors
    return err, true
}
```

Sentinel errors:
- `retry.ErrLimitExceeded`
- `retry.ErrThrottled`
- `retry.ErrCanceled`

### HTTP RoundTripper

```go
import "net/http"

cfg := retry.DefaultConfig()
cfg.Attempts = 3
cfg.Backoff = retry.NewExponentialBackoff(50*time.Millisecond, 500*time.Millisecond)
r := retry.New(cfg)

client := &http.Client{
    Transport: retry.NewRoundTripper(http.DefaultTransport, r),
    Timeout:   10 * time.Second,
}
```

`NewRoundTripper(rt, r)` uses `DefaultStatusCodeHandler` which retries on 408, 425, 500, 502, 503, 504. Provide a custom handler via `RoundTripper.StatusCodeHandler`.

The transport closes response bodies on retryable errors and recreates the request body via `GetBody` when present.

## API reference

* `DefaultConfig() *Config`
* `New(cfg *Config) *Retry`
* `(*Retry).Do(ctx, fn) error`
* `Func[K,V](fn, rt) func(ctx, K) (V, error)`
* `NewConstantBackoff(period)`, `NewLinearBackoff(period)`, `NewExponentialBackoff(base, cap)`
* `NewThrottler(cfg)`, `NewNoopThrottler()`
* `NewRoundTripper(rt, r) *RoundTripper`

## License

MIT
