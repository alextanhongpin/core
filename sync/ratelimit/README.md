# Rate Limiting Library

A small, thread-safe, key-based rate limiting library for Go. This package provides two classic algorithms — Fixed Window and GCRA — with a simple `Config` API and a `Result` type for introspection.

## Features

- **Key-based limiting** – each `Allow(key)` call is tracked per unique key (user ID, API key, etc.)
- **Two algorithms**
  - `FixedWindow` – simple counter reset each period, minimal memory
  - `GCRA` – Generic Cell Rate Algorithm, smooth traffic, burst support
- **Zero dependencies** – only Go standard library
- **Thread-safe** – `sync.RWMutex` for concurrent use
- **Result introspection** – `Allow`, `Remaining`, `ResetAfter`, `RetryAfter`, `Limit`

## Installation

```bash
go get github.com/alextanhongpin/core/sync/ratelimit
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    "github.com/alextanhongpin/core/sync/ratelimit"
)

func main() {
    cfg := ratelimit.Config{
        Limit:  5,
        Period: time.Second,
        Burst:  0,
    }
    // Fixed Window
    fw := ratelimit.NewFixedWindow(cfg)
    if fw.Allow("user-1") {
        fmt.Println("allowed")
    }

    // GCRA with burst
    cfgGCRA := ratelimit.Config{
        Limit:  5,
        Period: time.Second,
        Burst:  2,
    }
    gcra := ratelimit.NewGCRA(cfgGCRA)
    res := gcra.Limit("user-1")
    fmt.Printf("allow=%v remaining=%d reset_after=%v\n", res.Allow, res.Remaining, res.ResetAfter)
}
```

## API

### Config

```go
type Config struct {
    Limit  int           // >0
    Period time.Duration // >0
    Burst  int           // >=0, used by GCRA only
}
func (cfg Config) Validate() error
```

`Validate` checks `Limit > 0`, `Period > 0`, `Burst >= 0`.

### Constructors

```go
func NewFixedWindow(cfg Config) *FixedWindow
func NewGCRA(cfg Config) *GCRA
```

Both validate via `cfg.Validate()` and panic on error. See *Improvements* for error-return variants.

### Common interface

```go
type RateLimiter interface {
    Allow(key string) bool
    AllowN(key string, n int) bool
    Limit(key string) *Result
    LimitN(key string, n int) *Result
}
```

`Allow` is equivalent to `Limit(...).Allow`.

### Result

```go
type Result struct {
    Allow      bool
    Remaining  int
    ResetAfter time.Duration // time until window resets
    RetryAfter time.Duration // time until next request is allowed
    Limit      int
}
```

### FixedWindow

Simple counter per key that resets at period boundaries.

```go
cfg := ratelimit.Config{Limit: 100, Period: time.Minute}
fw := ratelimit.NewFixedWindow(cfg)

allowed := fw.Allow("api-key-123")
res := fw.Limit("api-key-123")
fmt.Println(res.Remaining, res.ResetAfter)
```

Memory: O(k) where k = number of unique keys. `Clear()` removes expired entries, `Size()` returns current key count.

### GCRA

Generic Cell Rate Algorithm provides smooth rate limiting with burst allowance.

```go
cfg := ratelimit.Config{Limit: 10, Period: time.Minute, Burst: 5}
rl := ratelimit.NewGCRA(cfg)

if rl.Allow("user-42") {
    // request permitted
}
```

Interval = `Period / Limit`. A request is allowed when `last - burst*interval <= now`. On success `last += interval`.

## Algorithm Details

### Fixed Window

1. On first request for a key, record `last = now`, `count = 0`
2. If `now >= last + period`, reset window: `last = now`, `count = 0`
3. If `count + n <= limit`, allow and `count += n`
4. `Remaining = limit - count`, `ResetAfter = last + period - now`

*Known limitation*: burst of requests at window boundaries. Consider GCRA for smoother traffic.

### GCRA

Tracks a virtual finish time `t` per key.
- `t = max(t, now)`
- Allow if `t - burst*interval <= now`
- On allow: `t += interval`

Provides excellent burst control and smooth distribution.

## Testing

Tests are data-driven via `github.com/alextanhongpin/evaltest`. Run:

```bash
go test -v
```

Example test input:

```yaml
name: valid allow
input:
  limit: 5
  period: 1s
  burst: 0
  key: user1
  action: Limit
evals:
  - expr: output.Allow == true
  - expr: output.Remaining == 4
```

## Improvements & Known Issues

The current implementation is functional but has several areas for improvement:

1. **Error handling** – constructors panic on invalid config. Prefer `NewFixedWindow(cfg) (*FixedWindow, error)` and `MustNewFixedWindow` for panicking variant.

2. **Fixed Window off-by-one** – current `LimitN` allows `limit+1` requests due to `<= limit+1` check and increments before checking allowance. Fix to check `count + n <= limit` before increment, and return `Allow = false` without mutating state when denied.

3. **Time source injection** – tests currently rely on real `time.Now()`. Add an optional `Now func() time.Time` field to allow deterministic testing.

4. **Burst parameter unused** – `FixedWindow` accepts `Burst` in `Config` but never uses it. Validate and ignore or document as no-op.

5. **Remaining calculation** – GCRA `Remaining` is approximate. Clarify semantics and ensure it matches `limit - used` for current window.

6. **Empty key handling** – `LimitN` returns empty `Result` for empty key. Consider returning explicit error.

7. **Memory cleanup** – `Clear()` iterates over all keys. For high cardinality keys, consider background eviction or sync.Map.

8. **Documentation** – README previously described Sliding Window, Multi-Key variants, metrics collectors, and old constructors that no longer exist. This README has been updated to match the current codebase.

9. **Observability** – Add optional metrics hooks for allowed/denied counts.

10. **API consistency** – Provide a single-key convenience wrapper or deprecate key parameter if only per-key limiting is needed.

## License

MIT – part of [alextanhongpin/core](https://github.com/alextanhongpin/core)
