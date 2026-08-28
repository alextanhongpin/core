# rate

A high-performance, thread-safe Go library for rate tracking and circuit-breaker-style limiting using exponential decay algorithms.

## Features

- **`Rate`** — Exponential decay rate counter for measuring events per time period
- **`Limiter`** — Token-based circuit breaker that blocks operations after accumulated failures
- **`Errors`** — Combined success/failure rate monitoring with time-based decay
- **Thread-safe** — All components use mutex-protected state for concurrent use
- **Testable** — Injectable `Now` function for deterministic, time-controlled tests

## Installation

```bash
go get github.com/alextanhongpin/core/sync/rate
```

## Components

### 1. `Rate` — Exponential Decay Rate Counter

Tracks the rate of events over a configurable time window using exponential smoothing. Old measurements decay automatically; no sliding window or ring buffer needed.

**Decay formula:** `count = count * (1 - elapsed/period) + n`

#### API

| Method | Description |
|---|---|
| `New() *Rate` | Creates a counter with a 1-second period |
| `Per(period time.Duration) *Rate` | Creates a counter with the given period (panics if ≤ 0) |
| `(r *Rate) Inc() float64` | Increments by 1, returns current smoothed count |
| `(r *Rate) Add(n float64) float64` | Adds `n`, returns current smoothed count |
| `(r *Rate) Count() float64` | Returns current smoothed count (calls `Add(0)`) |
| `(r *Rate) Per(t time.Duration) float64` | Scales current count to a different time unit |
| `(r *Rate) Reset()` | Resets the counter to zero |
| `(r *Rate) Now func() time.Time` | Injectable time function (assign directly for testing) |

#### Examples

```go
// Per-second and per-minute rate counters
rps := rate.Per(time.Second)
rpm := rate.Per(time.Minute)

rps.Inc()         // +1 event
rps.Add(5)        // +5 events

fmt.Println(rps.Count())              // current smoothed count (1-second basis)
fmt.Println(rps.Per(time.Minute))     // scale count to per-minute
```

```go
// HTTP request rate tracking
requestRate := rate.Per(time.Minute)

http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    current := requestRate.Inc()
    log.Printf("Rate: %.2f req/min", current)
    // handle request...
})
```

#### Behaviour at different time scales

The table below shows how three counters (1s, 1m, 1h period) evolve when one event arrives every 100 ms:

```
     ms|    s|     m|     h|
     0s| 1.00|  1.00|  1.00|
  100ms| 1.90|  2.00|  2.00|
  200ms| 2.71|  3.00|  3.00|
  300ms| 3.44|  3.99|  4.00|
  ...
```

(See `ExampleRate` in [`rate_test.go`](rate_test.go) for the full output across constant and exponential intervals.)

---

### 2. `Limiter` — Token-based Circuit Breaker

Accumulates *failure tokens* on errors and subtracts *success tokens* on success. Operations are blocked (returns `ErrLimitExceeded`) once tokens reach the configured limit. This implements a leaky-token circuit breaker without any time window.

#### API

| Method / Field | Description |
|---|---|
| `NewLimiter(limit float64) *Limiter` | Creates a limiter; panics if limit ≤ 0 |
| `FailureToken float64` | Tokens added per failure (default `1.0`) |
| `SuccessToken float64` | Tokens subtracted per success (default `0.5`) |
| `(l *Limiter) Allow() bool` | Returns `true` if current tokens < limit |
| `(l *Limiter) Do(fn func() error) error` | Checks limit, runs `fn`, records result; returns `ErrLimitExceeded` if blocked |
| `(l *Limiter) Err()` | Manually record a failure (adds `FailureToken`) |
| `(l *Limiter) Ok()` | Manually record a success (subtracts `SuccessToken`) |
| `(l *Limiter) Success() int` | Total successful operations |
| `(l *Limiter) Failure() int` | Total failed operations |
| `(l *Limiter) Total() int` | Total operations (success + failure) |

**Error sentinel:** `rate.ErrLimitExceeded`

#### Examples

```go
// Simple circuit breaker — open after 3 failure tokens
limiter := rate.NewLimiter(3)

err := limiter.Do(func() error {
    return callExternalService()
})
if errors.Is(err, rate.ErrLimitExceeded) {
    // circuit is open; shed the load
}
```

```go
// Manual token control
limiter := rate.NewLimiter(5)

if limiter.Allow() {
    if err := doWork(); err != nil {
        limiter.Err() // +1.0 token
    } else {
        limiter.Ok()  // -0.5 token
    }
}
```

```go
// Tuning token weights
aggressive := rate.NewLimiter(5)
aggressive.FailureToken = 2.0 // Opens faster
aggressive.SuccessToken = 1.0 // Recovers faster

conservative := rate.NewLimiter(20)
conservative.FailureToken = 0.5 // Opens slowly
conservative.SuccessToken = 1.0 // Recovers quickly
```

#### Token accumulation example

With a limit of 3 and default tokens (failure=1.0, success=0.5):

| Operation | Tokens | Allowed |
|---|---|---|
| start | 0.0 | ✅ |
| Err | 1.0 | ✅ |
| Err | 2.0 | ✅ |
| Err | 3.0 | ❌ (`ErrLimitExceeded`) |
| Ok (blocked) | 3.0 | ❌ |

---

### 3. `Errors` — Success/Failure Rate Tracker

Wraps two `Rate` counters (one for successes, one for failures) and provides error-ratio snapshots via `ErrorRate`.

#### API

| Method | Description |
|---|---|
| `NewErrors(period time.Duration) *Errors` | Creates tracker with given decay period |
| `(e *Errors) Success() counter` | Returns the success counter (`Inc`, `Add`, `Count`) |
| `(e *Errors) Failure() counter` | Returns the failure counter |
| `(e *Errors) Rate() *ErrorRate` | Snapshot of current success/failure rates |
| `(e *Errors) Reset()` | Resets both counters |
| `(e *Errors) SetNow(func() time.Time)` | Sets time function on both counters (for testing) |

**`ErrorRate` snapshot methods:**

| Method | Description |
|---|---|
| `Success() float64` | Smoothed success rate |
| `Failure() float64` | Smoothed failure rate |
| `Total() float64` | `Success + Failure` (float64) |
| `Ratio() float64` | `Failure / Total` — returns 0 if no events |

#### Examples

```go
tracker := rate.NewErrors(5 * time.Minute)

// Record events
tracker.Success().Inc()
tracker.Failure().Add(2)

// Snapshot
snap := tracker.Rate()
fmt.Printf("Success: %.2f/5min\n", snap.Success())
fmt.Printf("Failure: %.2f/5min\n", snap.Failure())
fmt.Printf("Error ratio: %.1f%%\n", snap.Ratio()*100)
```

```go
// Health-check helper
func isHealthy(tracker *rate.Errors) bool {
    h := tracker.Rate()
    // Need at least 10 events before deciding; < 10% error rate
    return h.Total() < 10 || h.Ratio() < 0.1
}
```

---

## Real-world Patterns

### Circuit Breaker for External APIs

```go
type APIClient struct {
    client  *http.Client
    breaker *rate.Limiter
}

func NewAPIClient() *APIClient {
    breaker := rate.NewLimiter(3) // open after 3 failure tokens
    breaker.SuccessToken = 1.0   // full recovery on each success
    return &APIClient{
        client:  &http.Client{Timeout: 5 * time.Second},
        breaker: breaker,
    }
}

func (c *APIClient) Get(ctx context.Context, url string) (*http.Response, error) {
    var resp *http.Response
    err := c.breaker.Do(func() error {
        req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
        var reqErr error
        resp, reqErr = c.client.Do(req)
        if reqErr != nil {
            return reqErr
        }
        if resp.StatusCode >= 500 {
            return fmt.Errorf("server error: %d", resp.StatusCode)
        }
        return nil
    })
    if errors.Is(err, rate.ErrLimitExceeded) {
        return nil, fmt.Errorf("circuit open: too many recent failures")
    }
    return resp, err
}
```

### Service Health Monitor

```go
type ServiceMonitor struct {
    mu      sync.RWMutex
    metrics map[string]*rate.Errors
}

func NewServiceMonitor() *ServiceMonitor {
    return &ServiceMonitor{metrics: make(map[string]*rate.Errors)}
}

func (sm *ServiceMonitor) Record(operation string, success bool) {
    sm.mu.Lock()
    if _, ok := sm.metrics[operation]; !ok {
        sm.metrics[operation] = rate.NewErrors(5 * time.Minute)
    }
    tracker := sm.metrics[operation]
    sm.mu.Unlock()

    if success {
        tracker.Success().Inc()
    } else {
        tracker.Failure().Inc()
    }
}

func (sm *ServiceMonitor) Status(operation string) string {
    sm.mu.RLock()
    tracker, ok := sm.metrics[operation]
    sm.mu.RUnlock()
    if !ok {
        return "UNKNOWN"
    }
    h := tracker.Rate()
    switch {
    case h.Total() < 5:
        return "UNKNOWN"
    case h.Ratio() < 0.05:
        return "HEALTHY"
    case h.Ratio() < 0.20:
        return "DEGRADED"
    default:
        return "UNHEALTHY"
    }
}
```

### HTTP Request Monitoring Middleware

```go
type Middleware struct {
    requests *rate.Rate
    errors   *rate.Rate
}

func NewMiddleware() *Middleware {
    return &Middleware{
        requests: rate.Per(time.Minute),
        errors:   rate.Per(time.Minute),
    }
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        m.requests.Inc()
        rw := &statusRecorder{ResponseWriter: w, code: 200}
        next.ServeHTTP(rw, r)
        if rw.code >= 400 {
            m.errors.Inc()
        }
        log.Printf("%s %s %d  req/min=%.2f  err/min=%.2f",
            r.Method, r.URL.Path, rw.code,
            m.requests.Count(), m.errors.Count())
    })
}

type statusRecorder struct {
    http.ResponseWriter
    code int
}

func (r *statusRecorder) WriteHeader(code int) {
    r.code = code
    r.ResponseWriter.WriteHeader(code)
}
```

### Adaptive Rate Limiter

```go
type AdaptiveLimiter struct {
    baseLimit    float64
    limiter      *rate.Limiter
    health       *rate.Errors
    mu           sync.Mutex
    lastAdjusted time.Time
}

func NewAdaptiveLimiter(baseLimit float64) *AdaptiveLimiter {
    return &AdaptiveLimiter{
        baseLimit:    baseLimit,
        limiter:      rate.NewLimiter(baseLimit),
        health:       rate.NewErrors(time.Minute),
        lastAdjusted: time.Now(),
    }
}

func (a *AdaptiveLimiter) Allow() bool {
    a.maybeAdjust()
    return a.limiter.Allow()
}

func (a *AdaptiveLimiter) maybeAdjust() {
    a.mu.Lock()
    defer a.mu.Unlock()
    if time.Since(a.lastAdjusted) < 30*time.Second {
        return
    }
    h := a.health.Rate()
    if h.Total() < 10 {
        return
    }
    var newLimit float64
    switch {
    case h.Ratio() > 0.20:
        newLimit = a.baseLimit * 0.50
    case h.Ratio() > 0.10:
        newLimit = a.baseLimit * 0.75
    case h.Ratio() < 0.05:
        newLimit = a.baseLimit * 1.25
    default:
        newLimit = a.baseLimit
    }
    a.limiter = rate.NewLimiter(newLimit)
    a.lastAdjusted = time.Now()
}
```

---

## Testing

All components support injectable time functions for deterministic, sleep-free tests.

### `Rate` — assign `Now` directly

```go
func TestRateDecay(t *testing.T) {
    now := time.Now()
    r := rate.Per(5 * time.Second)

    r.Now = func() time.Time { return now }
    assert.Equal(t, 1.0, r.Inc()) // t=0s

    r.Now = func() time.Time { return now.Add(1 * time.Second) }
    assert.Equal(t, 1.8, r.Inc()) // t=1s — 20% decay

    r.Now = func() time.Time { return now.Add(2 * time.Second) }
    assert.Equal(t, 2.44, r.Inc()) // t=2s
}
```

### `Errors` — use `SetNow`

```go
func TestErrorRatio(t *testing.T) {
    now := time.Now()
    tracker := rate.NewErrors(5 * time.Second)
    tracker.SetNow(func() time.Time { return now })

    tracker.Success().Add(1)
    tracker.Failure().Add(1)

    snap := tracker.Rate()
    assert.Equal(t, 0.5, snap.Ratio())
    assert.Equal(t, 1.0, snap.Success())
    assert.Equal(t, 1.0, snap.Failure())
}
```

### `Limiter` — fully synchronous, no time dependency

```go
func TestCircuitBreaker(t *testing.T) {
    is := assert.New(t)
    limiter := rate.NewLimiter(3)
    badErr := errors.New("fail")

    // 3 failures fill the token bucket
    for range 3 {
        is.ErrorIs(limiter.Do(func() error { return badErr }), badErr)
    }

    // Circuit is now open
    is.ErrorIs(limiter.Do(func() error { return nil }), rate.ErrLimitExceeded)
    is.Equal(3, limiter.Failure())
    is.Equal(0, limiter.Success())
    is.Equal(3, limiter.Total())
}
```

---

## Configuration Guide

### Rate counter periods

| Period | Use case |
|---|---|
| `time.Second` | Real-time dashboards, immediate alerting |
| `time.Minute` | Operational metrics, request-rate alarms |
| `5 * time.Minute` | Trend analysis, health checks |
| `time.Hour` | SLA tracking, capacity planning |

### Limiter token presets

| Preset | `FailureToken` | `SuccessToken` | Behaviour |
|---|---|---|---|
| Default | 1.0 | 0.5 | Moderate — opens after N failures, slow recovery |
| Aggressive | 2.0 | 1.0 | Opens quickly, recovers quickly |
| Conservative | 0.5 | 1.0 | Opens slowly, recovers quickly |
| Strict | 1.0 | 0.1 | Opens at moderate pace, very slow recovery |

---

## Performance Notes

- All operations are **O(1)** — exponential decay avoids ring buffers or sorted lists
- Memory is **constant** regardless of event volume or time window length
- Each operation acquires a mutex; for extremely high-throughput paths consider per-goroutine instances with periodic aggregation
- `Limiter.Do` uses a write lock for the gate check to prevent TOCTOU races

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `go test -v`
5. Submit a pull request

## License

MIT License — see LICENSE file for details.
