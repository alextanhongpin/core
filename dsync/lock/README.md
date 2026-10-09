# Redis leases

Coordinate keyed operations across processes with renewable Redis leases.

## Installation and documentation

```sh
go get github.com/alextanhongpin/core/dsync/lock
go doc github.com/alextanhongpin/core/dsync/lock
```

[Go API reference](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/lock) · [Module requirements](go.mod)

## Typical use

Use to coordinate scheduled refreshes or maintenance jobs sharing one Redis primary.

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
locker, err := lock.New(lock.NewClient(client), lock.DefaultConfig())
if err != nil {
	return err
}
return locker.Do(ctx, "reports:daily", func(ctx context.Context) error {
	return rebuildReport(ctx) // Stop promptly if ctx is canceled.
})
```

## Behavior and configuration

New(client, Config{}) returns a Locker and error. Configuration is passed by value, defaulted, validated, and stored privately. Zero LockTTL selects 30 seconds; zero RefreshRatio disables renewal. DefaultConfig explicitly enables renewal at ratio 0.8. Nil Retry selects a contention-only retry policy; nil Logger selects slog.Default. MustNew is the panicking startup helper.

Do admits same-instance callers by key with cancelable local admission, acquires a unique Redis token, and invokes the callback synchronously. On caller cancellation or lease loss it cancels the callback context and waits for callback completion before cleanup. Callbacks must cooperate with cancellation and must not reenter the same key. Panics propagate after cleanup; cleanup errors are joined with operation errors. Cleanup has a five-second timeout detached from caller cancellation.

Callbacks, retry runners, and Redis clients remain shared dependencies and must support concurrent use. Func returns the final callback value and error; resource-bearing callback results remain caller-owned. No instance owns the Redis client's lifetime.

NewClient uses token-checked SET NX acquisition and SETIFDEQ/DELEX renewal/release, requiring a Redis release that supports these commands. These are single-primary leases, not fencing: expiration, failover, or paused processes can permit overlapping external work. Use fencing at the protected resource when required.

## Expected errors

`ErrLocked` means acquisition or token ownership conflicted. `ErrExpired` means the lease disappeared. `ErrLockTimeout` limits work when renewal is disabled. The default contention retry can return `retry.ErrLimitExceeded`; use `errors.Is` for wrapped errors.

## Pitfalls

A lease can expire while external work continues. Callbacks must honor cancellation; use resource-side fencing when overlapping writes would be unsafe.
