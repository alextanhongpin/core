# Redis circuit breaker

Share breaker state and failure accounting across service instances through Redis.

## Installation and documentation

```sh
go get github.com/alextanhongpin/core/dsync/circuitbreaker
go doc github.com/alextanhongpin/core/dsync/circuitbreaker
```

[Go API reference](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/circuitbreaker) · [Module requirements](go.mod)

## Typical use

Use when multiple replicas call the same failing dependency and should share its recovery state. Scope keys by downstream service.

Examples assume a caller-owned Redis client and a context. Create the client once
and close it after all operations finish:

```go
client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
defer client.Close()
ctx := context.Background()
```

Import `context`, `github.com/redis/go-redis/v9`, and this package's import path.

## Quick start

The snippet belongs inside a function returning `error`; application functions
such as `handleRequest` represent your own work. Import the package above and
the standard packages used in the snippet (`context`, `errors`, `fmt`, or `time`).

```go
if err := circuitbreaker.Setup(ctx, client); err != nil {
	return err
}
breaker, err := circuitbreaker.New(client, circuitbreaker.Config{FailureThreshold: 3})
if err != nil {
	return err
}
err = breaker.Do(ctx, "downstream:catalog", func() error {
	return fetchCatalog(ctx)
})
if errors.Is(err, circuitbreaker.ErrOpened) {
	// Serve a cached response or reject this request.
}
return err
```

## Behavior and configuration

New(client, Config{}) returns a breaker and error. MustNew is the panicking startup helper. Configuration is passed by value and stored privately; the borrowed client and weighting callbacks remain shared and must support concurrent calls. Zero thresholds and durations select defaults. Durations must be at least one millisecond. Nil hooks select defaults; supply zero-returning hooks to disable extra weighting.

Run Setup on every Redis primary before use. It installs cb_begin, cb_commit, and cb_set_status functions. Deploy callers and functions together; Setup replaces the function library. Redis must support functions and hash-field expiration.

Do admits once, executes the callback synchronously, and rejects opened states with ErrOpened. Callbacks run without a local mutex and may execute concurrently. Failed calls contribute 1+FailureCount(err)+SlowCallCount(duration); successful slow calls do not count as failures. Weight hooks must be nonnegative. Half-open closes at the exact configured success threshold. It does not limit concurrent probes.

State transitions advance a Redis generation. Results admitted in an earlier generation cannot change a newer state. SetStatus(Opened) starts its timeout. Failed-operation commits use a five-second context detached from caller cancellation, preserving both operation and Redis errors. State keys remain until externally removed; deleting a key destroys generation history. Single-primary state does not promise continuity through Redis failover or data loss.

Func preserves the operation signature and caller context. With retry outside the breaker, each attempt contributes separately; outside retry, the breaker observes the logical operation once.

## Expected errors

`ErrOpened` means the callback did not run. Callback and Redis errors propagate and may be joined; use `errors.Is`. Invalid configuration fails at construction.

## Pitfalls

Run Setup before use. Redis errors are operational failures, and half-open state does not limit simultaneous probes.
