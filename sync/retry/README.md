# retry

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/sync/retry.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/sync/retry)
[![Go Report Card](https://goreportcard.com/badge/github.com/alextanhongpin/core/sync/retry)](https://goreportcard.com/report/github.com/alextanhongpin/core/sync/retry)

A robust, production-ready Go retry package providing configurable backoff strategies, adaptive client-side throttling (token bucket), context cancellation awareness, and HTTP `RoundTripper` integration.

## Features

- **Functional Decorator & Convenience APIs**: Generic middleware-style `Handler[K, V]`, as well as `Do` and `DoValue[T]` helpers.
- **Multiple Backoff Strategies**: Exponential backoff with full jitter, Linear backoff, Constant backoff, and immediate `NoWait`.
- **Adaptive Client-Side Throttling**: Token bucket limiter to prevent retry storms and cascading downstream failures.
- **Context-Aware Cancellation**: Immediate termination on context cancellation/timeout with leak-free timer management.
- **Granular Error Classification**: Define non-retryable errors (`WithNonRetryableErrors`) or dynamic predicates (`WithRetryable`).
- **HTTP Client Integration**: Drop-in `http.RoundTripper` with status code evaluation, automatic response body closing on retry, and request body rewinding (`GetBody`).

## Installation

```bash
go get github.com/alextanhongpin/core/sync/retry
```

## Quick Start

### 1. Basic Retry (`Do`)

Retry operations that return an `error`:

```go
package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func main() {
	ctx := context.Background()

	err := retry.Do(ctx, func(ctx context.Context) error {
		return performOperation(ctx)
	},
		retry.N(3),                                                // Up to 3 retries (4 attempts total)
		retry.Exponential(100*time.Millisecond, 2*time.Second),   // Jittered exponential backoff
	)
	if err != nil {
		log.Printf("Operation failed: %v", err)
	}
}

func performOperation(ctx context.Context) error {
	// Your network or database call
	return nil
}
```

### 2. Retry with Return Value (`DoValue`)

Retry operations that return both a value and an `error`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

type User struct {
	ID   string
	Name string
}

func main() {
	ctx := context.Background()

	user, err := retry.DoValue(ctx, func(ctx context.Context) (*User, error) {
		return fetchUser(ctx, "user-123")
	},
		retry.N(3),
		retry.Linear(50*time.Millisecond),
	)
	if err != nil {
		log.Fatalf("Failed to fetch user: %v", err)
	}

	fmt.Printf("User: %+v\n", user)
}

func fetchUser(ctx context.Context, id string) (*User, error) {
	return &User{ID: id, Name: "Alice"}, nil
}
```

### 3. Generic Function Decorator (`Handler`)

Wrap generic functions `func(ctx context.Context, req K) (V, error)` with retry middleware:

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

type GetItemRequest struct {
	ItemID string
}

type GetItemResponse struct {
	ItemName string
	Price    int
}

func getItem(ctx context.Context, req GetItemRequest) (GetItemResponse, error) {
	return GetItemResponse{ItemName: "Widget", Price: 100}, nil
}

func main() {
	// Wrap once and reuse across multiple invocations
	getItemWithRetry := retry.Handler(
		getItem,
		retry.N(3),
		retry.Exponential(50*time.Millisecond, 1*time.Second),
	)

	resp, err := getItemWithRetry(context.Background(), GetItemRequest{ItemID: "item-42"})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Item: %s, Price: $%d\n", resp.ItemName, resp.Price)
}
```

## Backoff Strategies

The package provides several backoff implementations conforming to the `Backoff` interface:

```go
type Backoff interface {
	At(attempts int) time.Duration
}
```

### 1. Exponential Backoff with Jitter (Recommended)

Applies full jitter between `0` and `min(Base * 2^attempts, Cap)` to prevent thundering herd problems:

```go
// Base duration 100ms, max duration 30s
retry.Exponential(100*time.Millisecond, 30*time.Second)
```

### 2. Linear Backoff

Linearly increasing delay calculated as `Period * attempts`:

```go
// 100ms * attempt (0ms, 100ms, 200ms, ...)
retry.Linear(100*time.Millisecond)
```

### 3. Constant Backoff & Immediate Retry

Fixed delay between attempts or zero delay:

```go
// Fixed 500ms delay
retry.Constant(500*time.Millisecond)

// No delay (immediate retry)
retry.NoWait
```

### 4. Custom Backoff

Implement the `retry.Backoff` interface:

```go
type FixedStepBackoff struct{}

func (b *FixedStepBackoff) At(attempts int) time.Duration {
	return time.Duration(attempts+1) * 200 * time.Millisecond
}

// Usage:
retry.WithBackoff(&FixedStepBackoff{})
```

## Adaptive Client-Side Throttling

When downstream services are under heavy load or failing, retries can exacerbate outages (retry storm). Adaptive throttling uses a **token bucket algorithm** to dynamically restrict retries when failures exceed a healthy ratio.

```go
// Enable default adaptive throttling (MaxTokens: 10, TokenRatio: 0.1)
retry.Throttle()

// Or configure custom throttler settings:
throttler := retry.NewThrottler(&retry.ThrottlerOptions{
	MaxTokens:  20,  // Maximum token capacity
	TokenRatio: 0.2, // Tokens replenished per successful request
})

err := retry.Do(ctx, fn,
	retry.N(5),
	retry.WithThrottler(throttler),
)
```

- Each retry consumes tokens from the bucket.
- Successful requests (including initial attempts and retries) replenish tokens via `TokenRatio`.
- If available tokens drop below threshold, retries are immediately aborted with `retry.ErrThrottled`, sparing downstream services.

## Error Handling & Classification

### Package Sentinel Errors

| Error | Description |
|---|---|
| `retry.ErrLimitExceeded` | Maximum retry attempts reached without success |
| `retry.ErrThrottled` | Retries aborted because token bucket is exhausted |
| `retry.ErrCanceled` | Operation encountered a non-retryable error or cancellation |

When multiple retries fail, `retry.Do` and `retry.Handler` return a joined error containing all attempted errors plus `ErrLimitExceeded`. Use standard `errors.Is` to check for specific conditions:

```go
if err != nil {
	switch {
	case errors.Is(err, retry.ErrLimitExceeded):
		log.Printf("All retry attempts exhausted: %v", err)
	case errors.Is(err, retry.ErrThrottled):
		log.Printf("Operation throttled to protect downstream service")
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		log.Printf("Context canceled or timed out: %v", err)
	default:
		log.Printf("Non-retryable error: %v", err)
	}
}
```

### Specifying Non-Retryable Errors

Prevent retrying fatal errors (e.g. `sql.ErrNoRows`, validation errors, 4xx client errors):

```go
var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid input")
)

err := retry.Do(ctx, fn,
	retry.N(3),
	retry.WithNonRetryableErrors(ErrNotFound, ErrInvalid),
)
```

### Dynamic Error Classification (`WithRetryable`)

Use a custom function to decide whether an error is retryable:

```go
err := retry.Do(ctx, fn,
	retry.N(3),
	retry.WithRetryable(func(err error) (error, bool) {
		// Do not retry context cancellation
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: %w", retry.ErrCanceled, err), false
		}

		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 429 {
			return err, true // Retry rate-limited requests
		}

		return err, false // Don't retry other errors
	}),
)
```

## HTTP Client Integration (`RoundTripper`)

The package includes a drop-in `http.RoundTripper` decorator that transparently retries failed HTTP requests:

- Automatically handles transient status codes (408, 425, 500, 502, 503, 504) by default.
- Properly closes response bodies from failed attempts to avoid connection/memory leaks.
- Supports replayable request payloads via `http.Request.GetBody`.

```go
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func main() {
	// Create an HTTP client with retry capabilities
	client := &http.Client{
		Transport: retry.NewRoundTripper(
			http.DefaultTransport,
			nil, // Defaults to standard transient status code handler
			retry.N(3),
			retry.Exponential(50*time.Millisecond, 500*time.Millisecond),
		),
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.example.com/data", nil)
	if err != nil {
		panic(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response (%d): %s\n", resp.StatusCode, string(body))
}
```

### Custom HTTP Status Code Handler

Customize which HTTP status codes trigger retries:

```go
customStatusCodeHandler := func(statusCode int) error {
	if statusCode == http.StatusTooManyRequests || statusCode >= 500 {
		return fmt.Errorf("retryable status code: %d", statusCode)
	}
	return nil
}

transport := retry.NewRoundTripper(
	http.DefaultTransport,
	customStatusCodeHandler,
	retry.N(5),
)
```

## Summary of Options

| Option | Description |
|---|---|
| `retry.WithAttempts(n)` / `retry.N(n)` | Number of retry attempts after initial attempt (default: `10`) |
| `retry.Exponential(base, cap)` | Jittered exponential backoff with base and max cap |
| `retry.Linear(period)` | Linear backoff scaling with attempt count |
| `retry.Constant(period)` | Constant delay between attempts |
| `retry.NoWait` | Zero-delay immediate retry |
| `retry.WithBackoff(b)` | Custom backoff strategy implementing `Backoff` |
| `retry.Throttle()` | Enable default token-bucket adaptive throttling |
| `retry.WithThrottler(t)` | Configure a custom `Limiter` throttler |
| `retry.WithNonRetryableErrors(errs...)` | Mark specific errors as permanent (non-retryable) |
| `retry.WithRetryable(fn)` | Custom predicate for error classification |

## License

MIT
