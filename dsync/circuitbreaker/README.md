# circuitbreaker

This package implements a **distributed circuit breaker** backed by Redis.
It is designed to protect services from repeatedly calling an unhealthy
dependency by short‑circuiting calls once a failure threshold is
exceeded.  The implementation is based on the standard circuit breaker
state machine (closed → open → half‑open → closed) and stores all
state in a Redis hash, so it works across multiple processes or
machines.

The core logic is implemented in Lua and loaded into Redis with the
`Setup` function.  The Go API then exposes a lightweight wrapper that
calls the Lua functions via `EVALSHA`.

## Features

* **Redis‑backed** – state is persisted in Redis, allowing multiple
  instances to coordinate.
* **Customisable thresholds** – failure and success thresholds,
  time windows, and open‑timeout are all configurable.
* **Fine‑grained failure counting** – callers can provide custom
  functions that map an error or call duration to a numeric failure
  count.
* **Convenient `Do` helper** – execute a function and let the circuit
  breaker decide whether to run it, record success or failure, and
  open/close the breaker automatically.
* **Force‑open / disable** – expose methods to manually set the state.

## Installation

```bash
go get github.com/alextanhongpin/core/dsync/circuitbreaker
```

The package requires a Redis server 6.2+ with Lua scripting enabled.

## Usage

### 1. Load the Lua script

The Lua script must be loaded into Redis before any circuit breaker
operations can take place.  The `Setup` helper does this for you.

```go
import (
    "context"
    redis "github.com/redis/go-redis/v9"
    "github.com/alextanhongpin/core/dsync/circuitbreaker"
)

ctx := context.Background()
client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

// Load the Lua script that implements the state machine.
if err := circuitbreaker.Setup(ctx, client); err != nil {
    panic(err)
}
```

### 2. Create a circuit breaker

```go
cb := circuitbreaker.New(client, nil) // use default options
```

Optionally, you can override the defaults:

```go
opts := circuitbreaker.NewOptions()
opts.FailureThreshold = 10
opts.OpenTimeout = 30 * time.Second
cb := circuitbreaker.New(client, opts)
```

### 3. Execute a protected call

```go
err := cb.Do(ctx, "my-service:db", func() error {
    // Call the real dependency.
    return callDatabase()
})

if err != nil {
    // The breaker either opened or the wrapped function returned an error.
}
```

If the breaker is **open** the `Do` method returns `circuitbreaker.ErrOpened`
immediately without invoking the wrapped function.

### 4. Inspect or override state

```go
status, _ := cb.Status(ctx, "my-service:db")
fmt.Println("current status:", status)

// Force‑open the breaker (e.g. during maintenance).
cb.SetStatus(ctx, "my-service:db", circuitbreaker.ForcedOpen)
```

## Real‑world examples

### 1. HTTP client with circuit breaker

```go
type HTTPClient struct {
    client *http.Client
    cb     *circuitbreaker.CircuitBreaker
}

func (h *HTTPClient) Get(url string) (*http.Response, error) {
    var resp *http.Response
    err := h.cb.Do(context.Background(), "http-client:backend", func() error {
        var err error
        resp, err = h.client.Get(url)
        return err
    })
    return resp, err
}
```

### 2. Database query with retry and circuit breaker

```go
func (s *Store) Query(ctx context.Context, q string) (*Result, error) {
    var res *Result
    err := s.cb.Do(ctx, "db:query", func() error {
        r, err := s.db.Query(ctx, q)
        res = r
        return err
    })
    return res, err
}
```

In both examples the breaker prevents thrashing the external
service once it becomes unreliable.

## Testing

Unit tests are provided in `circuitbreaker_test.go`.  They exercise
all state transitions and error handling.  Run them with:

```bash
go test ./...
```

## License

This project is licensed under the MIT License.  See the `LICENSE`
file for details.

