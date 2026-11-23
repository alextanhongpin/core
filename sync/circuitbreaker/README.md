# circuitbreaker

Stop calling a failing dependency for a while, then let a few probe calls
decide whether it has recovered. Includes adapters for functions and
`http.RoundTripper`.

**Use it for:** an unreliable downstream API or database call where failing
fast is better than piling up timeouts.

**Don't use it for:** state shared across processes (the breaker is
**in-process**; use [`dsync/circuitbreaker`](../../dsync/circuitbreaker), the
Redis-shared variant), limiting request rate ([`ratelimit`](../ratelimit)) or
bounding concurrency ([`throttle`](../throttle)).

## Installation

```sh
go get github.com/alextanhongpin/core/sync/circuitbreaker
```

[API reference](https://pkg.go.dev/github.com/alextanhongpin/core/sync/circuitbreaker)

## Quick start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
)

func main() {
	cfg := circuitbreaker.DefaultConfig()
	cfg.FailureThreshold = 3
	cfg.FailurePeriod = time.Second
	cfg.OpenTimeout = 5 * time.Second
	cb, err := circuitbreaker.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	boom := errors.New("dependency down")
	for i := 1; i <= 5; i++ {
		err := cb.Do(func() error { return boom })
		switch {
		case errors.Is(err, circuitbreaker.ErrOpened):
			fmt.Printf("call %d: rejected without running (status %s)\n", i, cb.Status())
		default:
			fmt.Printf("call %d: %v (status %s)\n", i, err, cb.Status())
		}
	}

	// Wrap an HTTP client: 5xx responses count as failures.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	client := &http.Client{Transport: circuitbreaker.NewTransporter(nil, circuitbreaker.MustNew(circuitbreaker.Config{}))}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	fmt.Println("http status:", resp.StatusCode)
}
```

Output:

```
call 1: dependency down (status closed)
call 2: dependency down (status closed)
call 3: dependency down (status opened)
call 4: rejected without running (status opened)
call 5: rejected without running (status opened)
http status: 200
```

## API

```go
func New(cfg Config) (*CircuitBreaker, error)
func MustNew(cfg Config) *CircuitBreaker // panics on invalid cfg
func DefaultConfig() Config
func (Config) WithDefaults() Config // zero fields select defaults
func (Config) Validate() error

type Config struct {
	FailureThreshold int           // weighted failures in one window that open the breaker
	FailurePeriod    time.Duration // fixed failure window
	SuccessThreshold int           // half-open successes in one window that close it
	SuccessPeriod    time.Duration // fixed success window
	OpenTimeout      time.Duration // time spent open before half-open
	HalfOpenMaxCalls int           // max concurrent half-open probes; 0 = unlimited
	FailureCount     func(cause error) int            // extra failure weight for an error
	SlowCallCount    func(duration time.Duration) int // extra failure weight for a slow failed call
}

func (*CircuitBreaker) Do(fn func() error) error // ErrOpened if rejected
func (*CircuitBreaker) Status() Status
func (*CircuitBreaker) SetStatus(Status)         // force a state; resets counters

func Func[K, V any](fn func(ctx context.Context, req K) (V, error), cb interface{ Do(func() error) error }) func(ctx context.Context, req K) (V, error)
func NewTransporter(rt http.RoundTripper, cb interface{ Do(func() error) error }) *Transporter // nil rt = http.DefaultTransport
func (*Transporter) RoundTrip(*http.Request) (*http.Response, error)

var ErrOpened error
const (Unknown, Closed, HalfOpen, Opened, Disabled, ForcedOpen Status)
```

## Behavior and guarantees

- **States.** Closed counts failures. When the weighted count reaches
  `FailureThreshold` within one `FailurePeriod`, it opens and rejects calls
  with `ErrOpened`. After `OpenTimeout` the next call moves it to half-open.
  Half-open closes after `SuccessThreshold` successes within one
  `SuccessPeriod`, and reopens on any failure. `Disabled` always runs `fn`;
  `ForcedOpen` always rejects.
- **Fixed windows.** A counting window starts at the first event after the
  previous window expired and ends `FailurePeriod`/`SuccessPeriod` later.
  Later events do not extend it, so a slow trickle of failures cannot keep a
  window alive forever.
- **Failure weight** is `1 + FailureCount(err) + SlowCallCount(duration)` for
  failed calls only. Successes in the closed state do not reset the failure
  window. Hooks must return non-negative values.
- **Half-open probes.** By default any number of calls run concurrently while
  half-open. Set `HalfOpenMaxCalls` to admit at most N in-flight probes; extra
  callers get `ErrOpened`. A probe slot is released when `fn` returns or
  panics. Negative values are rejected by `New`.
- **Concurrency.** Safe for concurrent use. `fn` and hooks run outside the
  lock. A result from a call admitted in an earlier state (before a transition
  or `SetStatus`) is ignored.
- **Errors.** `Do` returns `fn`'s error unchanged, or `ErrOpened` without
  running `fn`. The breaker does not recover panics.
- **`Func`** counts every error returned by the wrapped function, including
  `context.Canceled`; filter in the wrapped function if needed. With retry
  outside the breaker each attempt counts; with the breaker outside retry the
  whole logical operation counts once.
- **`Transporter`** counts transport errors and 5xx responses as failures,
  but not `context.Canceled` caused by the request context (the error is still
  returned). Such a call counts as neither success nor failure, so a canceled
  half-open probe cannot close the circuit. A discarded 5xx response
  is drained up to 4 KiB and closed so the connection can be reused. When the
  breaker rejects a request, the request body is closed.
- **Limitations.** State is per process and per instance; share one breaker
  per dependency. Counters are not decayed gradually (fixed windows), so
  failures straddling a window boundary can add up to nearly twice the
  threshold without opening.

## Testing

```sh
go vet ./... && go test -race -count=1 ./...
```

Some tests use the private `evaltest` module and golden files in `testdata/`.
