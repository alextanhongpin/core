# Circuit Breaker

A simple, idiomatic Go circuit breaker implementation with configurable thresholds and a minimal state machine.

## Features

- **Closed / Open / Half-Open** states with automatic transitions
- Configurable thresholds with sliding windows:
  - `FailureThreshold` and `FailurePeriod` (failures to open within period)
  - `SuccessThreshold` and `SuccessPeriod` (successes to close within period)
  - `OpenTimeout` (duration to wait before moving from Open to Half-Open)
- `FailureCount` and `SlowCallCount` hooks to penalize errors and slow calls
- Returns `ErrOpened` when calls are rejected
- Supports `Disabled` and `ForcedOpen` statuses for testing / manual control
- Context-aware `Do` API

## Installation

```bash
go get github.com/alextanhongpin/core/sync/circuitbreaker
```

## Basic Usage

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/alextanhongpin/core/sync/circuitbreaker"
)

func main() {
    cb := circuitbreaker.New(circuitbreaker.NewOptions())

    err := cb.Do(context.Background(), func(ctx context.Context) error {
        // simulate work or call to remote service
        return nil
    })

    if err == circuitbreaker.ErrOpened {
        fmt.Println("circuit is open, request short-circuited")
        return
    }

    if err != nil {
        fmt.Println("operation failed:", err)
        return
    }

    fmt.Println("operation succeeded")
}
```

## Handler

The `Handler` method decorates an existing function with circuit breaker capability. It returns a wrapped function that passes each call through the breaker:

```go
import (
    "context"
    "fmt"

    "github.com/alextanhongpin/core/sync/circuitbreaker"
)

func main() {
    cb := circuitbreaker.New(circuitbreaker.NewOptions())

    // Decorate an existing function with circuit breaker
    decoratedFn := cb.Handler(func(ctx context.Context, name string) (string, error) {
        // simulate work or call to remote service
        return "hello " + name, nil
    })

    // Call the decorated function - it will be protected by the circuit breaker
    result, err := decoratedFn(context.Background(), "world")
    if err != nil {
        fmt.Println("operation failed:", err)
        return
    }

    fmt.Println("result:", result)
}
```

The handler preserves the original function signature while adding circuit breaker protection. It's useful for wrapping existing APIs, middleware, or adapting third-party functions with resilience patterns.

## Configuration

Defaults are provided by `NewOptions`:

```go
opts := circuitbreaker.NewOptions()
opts.FailureThreshold = 100
opts.FailurePeriod    = time.Second
opts.SuccessThreshold = 20
opts.SuccessPeriod    = time.Second
opts.OpenTimeout      = time.Minute
opts.FailureCount = func(cause error) int {
    if cause != nil && errors.Is(cause, context.DeadlineExceeded) {
        return 2
    }
    return 0
}
opts.SlowCallCount = func(d time.Duration) int {
    if d >= time.Minute {
        return 4
    }
    if d >= 30*time.Second {
        return 2
    }
    if d > time.Second {
        return 1
    }
    return 0
}

cb := circuitbreaker.New(opts)
```

`New` accepts a nil options pointer and falls back to defaults:

```go
cb := circuitbreaker.New(nil)
```

### State control

```go
cb.SetStatus(circuitbreaker.Opened)
status := cb.Status() // circuitbreaker.Status
```

Statuses:

- `Unknown`
- `Closed`
- `HalfOpen`
- `Opened`
- `Disabled`
- `ForcedOpen`

## State Machine

1. **Closed**: all calls pass; failures and slow calls increment a TTL counter that expires after `FailurePeriod`.
2. **Opened**: calls immediately reject with `ErrOpened`; after `OpenTimeout` expires, the breaker transitions to Half-Open on the next call.
3. **Half-Open**: allows probe calls; successes increment a TTL counter that expires after `SuccessPeriod`. Once `SuccessThreshold` successes are accumulated within the period, the breaker closes. A failure immediately reopens.

`Disabled` bypasses the breaker and always executes the function. `ForcedOpen` is equivalent to `Opened` but stays open indefinitely until manually changed.

## HTTP Transport Integration

Use the provided `Transporter` to wrap any HTTP client:

```go
import "net/http"

client := &http.Client{}
cb := circuitbreaker.New(circuitbreaker.NewOptions())

client.Transport = circuitbreaker.NewTransporter(client.Transport, cb)

// Now all HTTP requests will go through the circuit breaker
resp, err := client.Get("https://api.example.com/users")
```

`Transporter.RoundTrip` calls `cb.Do` with the request context and returns `ErrOpened` when the circuit is open.

## Testing

The breaker uses real time via `time.Now` for timeouts and TTL expiry. In tests, control time by adjusting the options and using `time.Sleep` or by setting short thresholds:

```go
opts := circuitbreaker.NewOptions()
opts.FailureThreshold = 5
opts.SuccessThreshold = 3
opts.OpenTimeout      = 100 * time.Millisecond
cb := circuitbreaker.New(opts)
```

## License

MIT License. See [LICENSE](../../LICENSE) for details.
