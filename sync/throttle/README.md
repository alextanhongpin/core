# Concurrency throttle

Bound concurrent operations and optionally queue a limited backlog.

## Installation and documentation

```sh
go get github.com/alextanhongpin/core/sync/throttle
go doc github.com/alextanhongpin/core/sync/throttle
```

[Go API reference](https://pkg.go.dev/github.com/alextanhongpin/core/sync/throttle) · [Module requirements](go.mod)

## Typical use

Use to protect a connection pool, CPU-intensive task, or downstream API from too many simultaneous calls.

## Quick start

The snippet belongs inside a function returning `error`; application functions
such as `handleRequest` represent your own work. Import the package above and
the standard packages used in the snippet (`context`, `errors`, `fmt`, or `time`).

```go
t, err := throttle.New(throttle.Config{
	Limit: 8, BacklogLimit: 16, BacklogTimeout: time.Second,
})
if err != nil {
	return err
}
err = t.Do(ctx, func(ctx context.Context) error {
	return sendReport(ctx) // Your operation must honor ctx.
})
if errors.Is(err, throttle.ErrCapacityExceeded) {
	// Reject or reschedule the work.
}
return err
```

## Behavior and configuration

`New(Config{}) (*Throttler, error)` uses a default concurrency limit of 1000, no backlog, and no admission waiting. `DefaultConfig()` explicitly enables 100 queued calls and a ten-second admission timeout. `MustNew` panics on invalid configuration for startup wiring.

Configuration is passed by value and stored privately. WithDefaults preserves zero backlog and timeout. Validate rejects negative settings, nonpositive effective limits, and total-capacity overflow.

Do executes the callback synchronously after admission, or rejects without calling it. It returns ErrCapacityExceeded when all running and backlog slots are occupied. BacklogTimeout limits admission waiting only; the callback receives the original caller context. Callbacks may execute concurrently and must cooperate with cancellation. Admission order is not FIFO. Permits are returned even if the callback panics.

Func preserves the context/request/result signature. Outside retry, one permit covers attempts and backoff; inside retry, each attempt obtains its own permit.

## Expected errors

`ErrCapacityExceeded` means running and backlog slots are full. `ErrTimeout` means admission waiting expired. Caller cancellation and callback errors propagate. Use `errors.Is` to distinguish them.

## Pitfalls

BacklogTimeout covers waiting for a permit, not execution. Give the callback a caller context with its own deadline.
