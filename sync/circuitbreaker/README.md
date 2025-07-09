# Circuit Breaker

A simple, idiomatic Go circuit breaker implementation with pluggable timing and hooks for observability.

## Features

- **Closed / Open / Half-Open** states with automatic transitions
- Configurable error and slow-call thresholds:
  - `FailureThreshold` (minimum failures)
  - `FailureRatio` (percentage of failures)
  - `SlowCallCount` penalty based on call duration
  - `SuccessThreshold` (minimum successes in half-open)
- Pluggable clock (`Now`) and timer (`AfterFunc`) for deterministic testing
- `OnStateChange` hook for logging, metrics, or custom reactions
- Returns `ErrBrokenCircuit` when calls are rejected
- Ignores context cancellations and applies heavier penalties on deadlines

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
    cb := circuitbreaker.New()

    // sample operation
    err := cb.Do(func() error {
        // simulate work or call to remote service
        return nil
    })

    if err == circuitbreaker.ErrBrokenCircuit {
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

## 🚀 Examples

### Simple Example

A basic example showing circuit breaker functionality:

```bash
go run examples/simple/main.go
```

### Advanced Example

A comprehensive example with metrics, callbacks, and multiple failure scenarios:

```bash
go run examples/main.go
```

### HTTP Client Example

Real-world HTTP client integration with circuit breaker:

```bash
go run examples/http/main.go
```

### HTTP Transport Integration

Use the provided `Transporter` to wrap any HTTP client:

```go
client := &http.Client{}
cb := circuitbreaker.New()
client.Transport = circuitbreaker.NewTransporter(client.Transport, cb)

// Now all HTTP requests will go through the circuit breaker
resp, err := client.Get("https://api.example.com/users")
```

## Advanced Configuration

You can customize defaults by modifying struct fields directly:

```go
cb := &circuitbreaker.Breaker{
    BreakDuration:    10 * time.Second,
    FailureThreshold: 20,
    FailureRatio:     0.25,
    SamplingDuration: 30 * time.Second,
    SuccessThreshold: 3,
    // penalize 1 error, +5 penalty for context deadlines
    FailureCount: func(err error) int { /* ... */ },
    // 1 penalty per full 2s of latency
    SlowCallCount: func(d time.Duration) int { return int(d/2*time.Second) },
    // hook called on every state change
    OnStateChange: func(from, to circuitbreaker.Status) {
        fmt.Printf("state: %s -> %s\n", from, to)
    },
}
```

## State Machine

1. **Closed**: all calls pass; failures and slow calls are counted.
2. **Open**: calls immediately reject with `ErrBrokenCircuit`; after `BreakDuration`, transitions to half-open.
3. **Half-Open**: allows exactly one probe call; if it succeeds and thresholds pass, closes; otherwise reopens.

## Metrics & Observability

The circuit breaker supports pluggable metrics collectors for tracking state transitions, failures, and recoveries. You can use the built-in atomic collector for in-memory stats, or integrate with Prometheus for production monitoring.

### Using the Atomic Metrics Collector (default)

By default, if you do not provide a metrics collector, an atomic in-memory collector is used:

```go
cb := circuitbreaker.New() // uses AtomicMetricsCollector by default
```

### Using Prometheus for Metrics

To collect metrics with Prometheus, inject a `PrometheusMetricsCollector`:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/alextanhongpin/core/sync/circuitbreaker"
)

stateChanges := prometheus.NewCounter(prometheus.CounterOpts{Name: "cb_state_changes", Help: "Circuit breaker state changes."})
opened := prometheus.NewCounter(prometheus.CounterOpts{Name: "cb_opened", Help: "Circuit opened events."})
closed := prometheus.NewCounter(prometheus.CounterOpts{Name: "cb_closed", Help: "Circuit closed events."})
prometheus.MustRegister(stateChanges, opened, closed)

metrics := &circuitbreaker.PrometheusMetricsCollector{
    StateChanges: stateChanges,
    Opened:       opened,
    Closed:       closed,
}
cb := circuitbreaker.New(circuitbreaker.WithMetrics(metrics))
```

## Testing

By injecting a custom `Now` and `AfterFunc`, you can advance or freeze time in tests, and assert transitions via `OnStateChange`.

```go
cb := circuitbreaker.New()
cb.Now = func() time.Time { return startTime }
cb.AfterFunc = func(d time.Duration, fn func()) *time.Timer {
    // manual trigger or fake timer
}
```

## License

MIT License. See [LICENSE](../../LICENSE) for details.
