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
- `Func` generic helper to wrap typed `func(ctx, req) (res, error)` functions
- HTTP `Transporter` integration that treats 5xx responses as failures

## Installation

```bash
go get github.com/alextanhongpin/core/sync/circuitbreaker
```

## Basic Usage

```go
import (
    "fmt"

    "github.com/alextanhongpin/core/sync/circuitbreaker"
)

func main() {
    cb := circuitbreaker.New(circuitbreaker.DefaultConfig())

    err := cb.Do(func() error {
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

## Func Helper

`Func` wraps a typed `func(ctx context.Context, req K) (V, error)` with circuit breaker protection, preserving the original signature:

```go
import (
    "context"
    "fmt"

    "github.com/alextanhongpin/core/sync/circuitbreaker"
)

func main() {
    cb := circuitbreaker.New(circuitbreaker.DefaultConfig())

    // Wrap a typed function with circuit breaker protection.
    protected := circuitbreaker.Func(func(ctx context.Context, name string) (string, error) {
        // simulate work or call to remote service
        return "hello " + name, nil
    }, cb)

    result, err := protected(context.Background(), "world")
    if err != nil {
        fmt.Println("operation failed:", err)
        return
    }

    fmt.Println("result:", result)
}
```

`Func` is useful for wrapping existing APIs or adapting third-party functions with resilience patterns without changing their call sites.

## Configuration

Defaults are provided by `DefaultConfig()`:

```go
opts := circuitbreaker.DefaultConfig()
opts.FailureThreshold = 100
opts.FailurePeriod    = time.Second
opts.SuccessThreshold = 20
opts.SuccessPeriod    = time.Second
opts.OpenTimeout      = time.Minute
opts.FailureCount = func(cause error) int {
    // Extra failure weight for deadline-exceeded errors.
    if errors.Is(cause, context.DeadlineExceeded) {
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

Config fields are public and can be mutated directly after construction:

```go
cb := circuitbreaker.New(nil)
cb.OpenTimeout = 100 * time.Millisecond
cb.FailureThreshold = 10
```

### How counters work

Each failure increments the failure counter by `1 + FailureCount(err) + SlowCallCount(duration)`, allowing high-severity errors (e.g. timeouts, slow calls) to count for more than one failure. Counters use a sliding TTL window that resets after `FailurePeriod` (or `SuccessPeriod` in Half-Open).

### State control

```go
cb.SetStatus(circuitbreaker.Opened)
status := cb.Status() // circuitbreaker.Status
```

Parse a status string (e.g. from config or persistence):

```go
status := circuitbreaker.ParseStatus("half-open") // circuitbreaker.HalfOpen
```

Statuses:

| Constant     | String          | Description                                      |
|--------------|-----------------|--------------------------------------------------|
| `Unknown`    | `"unknown"`     | Zero value; not normally used at runtime         |
| `Closed`     | `"closed"`      | Normal operation — all calls pass through        |
| `HalfOpen`   | `"half-open"`   | Probe mode — allows limited calls after timeout  |
| `Opened`     | `"opened"`      | Tripped — calls rejected with `ErrOpened`        |
| `Disabled`   | `"disabled"`    | Bypass mode — breaker logic is skipped entirely  |
| `ForcedOpen` | `"forced-open"` | Permanently open until manually changed          |

## State Machine

```
        failures >= FailureThreshold
  Closed ──────────────────────────────► Opened
    ▲                                      │
    │                                      │ OpenTimeout elapsed
    │ successes >= SuccessThreshold        ▼
    └──────────────────────────────── HalfOpen
                                           │
                                           │ any failure
                                           └──────────────► Opened
```

1. **Closed**: All calls pass through. Each failure increments a TTL counter (reset after `FailurePeriod`). When the counter reaches `FailureThreshold`, the breaker opens.
2. **Opened**: Calls are immediately rejected with `ErrOpened`. After `OpenTimeout` elapses, the next call transitions the breaker to Half-Open.
3. **Half-Open**: Probe calls are allowed. Successes increment a TTL counter (reset after `SuccessPeriod`). Once `SuccessThreshold` successes accumulate, the breaker closes. Any failure immediately reopens.

`Disabled` bypasses the breaker and always executes the function. `ForcedOpen` is equivalent to `Opened` but stays open indefinitely until manually changed.

## HTTP Transport Integration

Use the provided `Transporter` to wrap any HTTP client. It treats HTTP 5xx responses as failures in addition to transport-level errors:

```go
import "net/http"

client := &http.Client{}
cb := circuitbreaker.New(circuitbreaker.DefaultConfig())

client.Transport = circuitbreaker.NewTransporter(client.Transport, cb)

// Now all HTTP requests will go through the circuit breaker.
// 5xx responses and network errors both count as failures.
resp, err := client.Get("https://api.example.com/users")
if err == circuitbreaker.ErrOpened {
    // Circuit is open — request was not sent.
}
```

`Transporter.RoundTrip` wraps the underlying `RoundTripper` in `cb.Do` and returns `ErrOpened` when the circuit is open.

## Testing

The breaker uses real time via `time.Now` for timeouts and TTL expiry. In tests, use short thresholds. For deterministic time control, wrap tests in `synctest.Test` from the standard library's `testing/synctest` package (Go 1.24+):

```go
import "testing/synctest"

func TestBreaker(t *testing.T) {
    synctest.Test(t, func(t *testing.T) {
        opts := circuitbreaker.DefaultConfig()
        opts.FailureThreshold = 3
        opts.SuccessThreshold = 3
        opts.OpenTimeout = 100 * time.Millisecond
        cb := circuitbreaker.New(opts)

        // ... drive the breaker through states
        time.Sleep(cb.OpenTimeout) // synthetic time inside synctest
    })
}
```

## License

MIT License. See [LICENSE](../../LICENSE) for details.
