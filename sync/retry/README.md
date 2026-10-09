# retry

[Go reference](https://pkg.go.dev/github.com/alextanhongpin/core/sync/retry)

Retry synchronous operations with configurable backoff, cancellation, and a shared adaptive retry budget. The module's Go version is specified in `go.mod`.

```sh
go get github.com/alextanhongpin/core/sync/retry
```

## Typical use

Use for transient network failures, reads from an occasionally unavailable service,
or writes protected by an application idempotency key. Set a deadline for the whole
operation and classify validation or authorization failures as permanent.

Use `go doc github.com/alextanhongpin/core/sync/retry` for local API documentation.

## Basic usage

```go
r, err := retry.New(retry.Config{
    MaxRetries: 3, // Four calls at most: one initial call and three retries.
    Backoff: retry.NewExponentialBackoff(100*time.Millisecond, 2*time.Second),
})
if err != nil {
    return err
}
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
return r.Do(ctx, performOperation)
```

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

`Backoff.At(1)` specifies the first retry delay. Built-in strategies are:

- `NewConstantBackoff(period)`: constant delay; nonpositive periods produce zero.
- `NewLinearBackoff(period)`: `period * retryNumber`, saturated at the maximum duration.
- `NewExponentialBackoff(base, cap)`: full jitter in `[0, min(base * 2^retryNumber, cap))`. The first retry has an upper bound of `2 * base`. Nonpositive base or cap produces zero.

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
    Backoff: retry.NewExponentialBackoff(50*time.Millisecond, 500*time.Millisecond),
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

Eligible requests retry transport errors and responses with status 408, 425, 500, 502, 503, or 504. `WithStatusCodeHandler(func(int) error)` replaces status classification. `Retry-After` is preserved in the final response but does not schedule backoff automatically.

Discarded response bodies are closed before another attempt. When status retries stop, the adapter returns the final response with **nil error**, preserving its status, headers, and readable body. The caller must inspect the status and close that body. Cancellation, body-factory failures, and transport failures return errors. The original request body is closed even when the runner rejects before invocation.

A nil transport uses `http.DefaultTransport`; a nil runner creates a retrier with `DefaultConfig()`. Options apply at construction. Supplied transports, runners, and policy callbacks must support concurrent calls.

Runnable examples and behavioral tests live alongside the implementation.
