# retry

Retry synchronous operations with configurable backoff, cancellation, and a
shared adaptive retry budget, plus an `http.RoundTripper` adapter.

**Use it for:** transient network failures, reads from an occasionally
unavailable service, or writes protected by an application idempotency key.
Set a deadline for the whole operation and classify validation or authorization
failures as permanent.

**Don't use it for:** non-idempotent side effects without an idempotency
guarantee, or for choosing delays only (use [`sync/backoff`](../backoff)). For
cross-process mutual exclusion see [`dsync/lock`](../../dsync/lock).

## Installation

```sh
go get github.com/alextanhongpin/core/sync/retry
```

Check the module's `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/backoff"
	"github.com/alextanhongpin/core/sync/retry"
)

func main() {
	r, err := retry.New(retry.Config{
		MaxRetries: 3, // four calls at most: one initial call and three retries
		Backoff:    backoff.NewExponential(10*time.Millisecond, 100*time.Millisecond),
	})
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	calls := 0
	err = r.Do(ctx, func(context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("temporary")
		}
		return nil
	})
	fmt.Println(calls, err)
}
```

It prints `3 <nil>`.

## API

```go
func New(cfg Config) (*Retry, error)
func DefaultConfig() Config // ten retries, full-jitter exponential backoff, adaptive throttler
func (*Retry) Do(ctx context.Context, fn func(context.Context) error) error
func Func[K, V any](fn func(context.Context, K) (V, error), r Runner) func(context.Context, K) (V, error)
type Runner interface{ Do(context.Context, func(context.Context) error) error }

type Config struct {
	MaxRetries int                         // excludes the initial call
	Backoff    Backoff                     // alias of backoff.Backoff
	Retryable  func(error) (error, bool)   // classify; nil retries all but cancellation
	Throttler  Limiter
}
func NonRetryableErrors(targets ...error) func(error) (error, bool)

var ErrLimitExceeded, ErrThrottled, ErrCanceled error

func NewThrottler(ThrottlerConfig) (*Throttler, error)
func NewNoopThrottler() *NoopThrottler

func NewRoundTripper(transport http.RoundTripper, runner Runner, options ...RoundTripperOption) *RoundTripper
func WithStatusCodeHandler(fn func(int) error) RoundTripperOption
func WithRetryableRequest(fn func(*http.Request) bool) RoundTripperOption
func WithRetryAfter(maxWait time.Duration) RoundTripperOption // opt in to Retry-After
func DefaultStatusCodeHandler(code int) error
```

The old strategy names (`NewExponentialBackoff` and friends) remain as
deprecated forwards to `sync/backoff`.

## Behavior and guarantees

Use `go doc github.com/alextanhongpin/core/sync/retry` for full API documentation.

`New(Config{})` performs one call. Zero `MaxRetries` disables retries; negative values return a construction error. Omitted dependencies receive defaults. `DefaultConfig()` enables ten retries, with full-jitter exponential backoff and an adaptive throttler; throttling or cancellation may stop it earlier.

Configuration is copied and stored privately. Dependencies and callbacks remain shared and must support concurrent calls if the retrier is shared. Mutating the original config's fields does not reconfigure a retrier; construct another instance. Do not mutate backoff structs or callback state concurrently with execution. Construct `Retry` with `New`; its zero value is not usable.

## Classification and errors

The default policy retries all errors except wrapped or joined `context.Canceled` and `context.DeadlineExceeded`. Configure permanent failures and retry only operations whose side effects are safe to repeat:

```go
cfg := retry.DefaultConfig()
cfg.Retryable = retry.NonRetryableErrors(ErrInvalidInput, ErrNotFound)
r, err := retry.New(cfg)
```

`NonRetryableErrors` preserves the operation error and checks each target with `errors.Is(err, target)`. It copies the target slice. Custom policies return `(classifiedError, retryable)`; classified errors are retained, with the original error used if classification returns nil. Context cancellation is always terminal.

Use `errors.Is` to inspect:

- `ErrLimitExceeded`: the retry count was exhausted; wraps the last classified error.
- `ErrThrottled`: the shared retry budget rejected another attempt; wraps the last classified error.
- `ErrCanceled`: a context cancellation or deadline stopped execution; wraps the context error and any custom cancellation cause. Ordinary permanent errors do not receive this label.

Cancellation is checked before initial invocation, retry admission, and invocation after backoff. Callbacks must cooperate with context cancellation; the package cannot interrupt running callbacks. Cancellation racing with invocation can still allow that invocation. A successful callback returns success.

## Functions and composition

```go
wrapped := retry.Func(func(ctx context.Context, id string) (User, error) {
    return lookupUser(ctx, id)
}, r)
user, err := wrapped(ctx, "123")
```

`Func` preserves `func(context.Context, K) (V, error)`. It returns the last callback result and final runner error; rejection before the first invocation returns the zero result. Callers own cleanup of resource-bearing results from every attempt. Inputs are reused, so callbacks must not consume streams or mutate inputs in ways that prevent replay.

`Runner` implementations must execute callbacks synchronously and sequentially and finish them before `Do` returns. Concurrent calls of the returned function have separate result variables; shared dependencies and inputs still need concurrency safety.

Decorator order affects behavior. A limiter inside retry applies to every attempt; a limiter outside retry applies once to the logical operation and may hold its permit during backoff. An outer timeout includes every attempt and wait. Put logical-operation metrics outside retry and attempt metrics inside it. Avoid nested retry layers that multiply attempts.

## Backoff and throttling

Import `github.com/alextanhongpin/core/sync/backoff` for delay strategies. The old retry strategy names remain available as deprecated aliases and forwarding constructors.

`Backoff.At(1)` specifies the first retry delay. Built-in strategies are:

- `backoff.NewConstant(period)`: constant delay; nonpositive periods produce zero.
- `backoff.NewLinear(period)`: `period * retryNumber`, saturated at the maximum duration.
- `backoff.NewExponential(base, cap)`: full jitter in `[0, min(base * 2^retryNumber, cap))`. The first retry has an upper bound of `2 * base`. Nonpositive base or cap produces zero.

Backoff and callbacks run outside the throttler lock.

The adaptive throttler starts with ten tokens, permits retry admission while tokens exceed five, and consumes one token per admitted retry. Every successful operation, including an initial attempt, replenishes `0.1` tokens. It does not refill over time. A continuously failing operation therefore gets at most five retries with default settings. Initial attempts always remain available, allowing recovery to replenish the shared budget. Tokens consumed before a canceled wait are not refunded.

```go
limiter, err := retry.NewThrottler(retry.DefaultThrottlerConfig())
// Share limiter across retriers for one downstream service.
cfg.Throttler = limiter
// Or disable throttling explicitly:
cfg.Throttler = retry.NewNoopThrottler()
```

`ThrottlerConfig.MaxTokens == 0` defaults to ten; `TokenRatio == 0` disables replenishment. Negative and non-finite settings return errors. `Throttler` supports concurrent calls.

## HTTP

```go
r, err := retry.New(retry.Config{
    MaxRetries: 3,
    Backoff: backoff.NewExponential(50*time.Millisecond, 500*time.Millisecond),
})
if err != nil {
    return err
}
client := &http.Client{
    Transport: retry.NewRoundTripper(http.DefaultTransport, r),
    Timeout: 10*time.Second,
}
```

The default policy permits retries for GET, HEAD, OPTIONS, TRACE, PUT, and DELETE. Bodies must also be replayable through `Request.GetBody` (or absent). A non-replayable body gets one transport call, regardless of policy. The first attempt uses the original body; subsequent attempts use fresh bodies. Attempt requests are cloned with the runner's context.

POST and other mutations require explicit opt-in backed by a server idempotency guarantee:

```go
transport := retry.NewRoundTripper(http.DefaultTransport, r,
    retry.WithRetryableRequest(func(req *http.Request) bool {
        // This client only calls endpoints that enforce this key.
        return req.Header.Get("Idempotency-Key") != ""
    }),
)
```

The request policy replaces the default method policy. Generate an idempotency key once per logical operation, before retries.

Eligible requests retry transport errors and responses with status 408, 425, 500, 502, 503, or 504. `WithStatusCodeHandler(func(int) error)` replaces status classification. **429 is not retryable by default**; opt in with a handler that returns an error for it. By default `Retry-After` is preserved in the final response but does not schedule backoff.

Opt in to honoring it with `WithRetryAfter(maxWait)`:

```go
transport := retry.NewRoundTripper(http.DefaultTransport, r,
    retry.WithStatusCodeHandler(func(code int) error {
        if code == http.StatusTooManyRequests || code == http.StatusServiceUnavailable {
            return fmt.Errorf("status code: %d", code)
        }
        return nil
    }),
    retry.WithRetryAfter(30*time.Second), // never wait longer than this for one header
)
```

For a response about to be retried, the header may be delta-seconds or an HTTP-date. The wait before the next attempt is `max(backoff, Retry-After)`, with the header value capped at `maxWait`. Missing, malformed, negative, or past values add no wait; a nonpositive `maxWait` leaves the option off. The extra wait happens inside the attempt after the runner's backoff, counts against the request context, and returns a cancellation error if the context ends. Exhaustion or throttling returns the last response immediately, without waiting.

When a retryable response is discarded for another attempt, up to 4 KiB of its body is drained and it is then closed (so the connection can be reused). That body stays open until the next attempt starts. When status retries stop, the adapter returns the final response with **nil error**, preserving its status, headers, and readable body. This also holds when the throttler stops retries: the last response is returned with a nil error, like normal exhaustion. The caller must inspect the status and close that body. Cancellation, body-factory failures, and transport failures return errors. The original request body is closed even when the runner rejects before invocation.

A nil transport uses `http.DefaultTransport`; a nil runner creates a retrier with `DefaultConfig()`. Options apply at construction. Supplied transports, runners, and policy callbacks must support concurrent calls.

Runnable examples and behavioral tests live alongside the implementation.

## Testing

```sh
go vet ./...
go test -race -count=1 ./...
go test -count=1 ./...
```
