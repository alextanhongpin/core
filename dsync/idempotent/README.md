# Redis-backed idempotent tasks

Retain request/result pairs and coordinate repeated requests by key.

## Installation and documentation

```sh
go get github.com/alextanhongpin/core/dsync/idempotent
go doc github.com/alextanhongpin/core/dsync/idempotent
```

[Go API reference](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/idempotent) · [Module requirements](go.mod)

## Typical use

Use for repeated job submissions or API requests with a stable idempotency key. Include the tenant and operation in the key.

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
i, err := idempotent.NewWithRedis(client)
if err != nil {
	return err
}
handler, err := i.HandlerFunc(func(ctx context.Context, name string) (string, error) {
	return "Hello, " + name, nil
}, idempotent.HandlerConfig{KeepTTL: 24 * time.Hour})
if err != nil {
	return err
}
value, reused, err := handler.Do(ctx, "tenant:1:greeting:request-123", "Ada")
if err != nil {
	return err
}
fmt.Println(value, reused) // Repeat the same key/request to reuse the result.
```

## Behavior and configuration

New(client) and NewWithRedis(redisClient) return an Idempotent and error. Clients remain borrowed. HandlerFunc(fn, HandlerConfig{}) and Handler(task, HandlerConfig{}) return a handler and error. Configuration is passed by value and captured privately. Zero LockTTL and KeepTTL select ten seconds and 24 hours; TTLs below one millisecond are rejected. WithDefaults returns a copy and Validate does not mutate it. MustNew, MustNewWithRedis, MustHandler, and MustHandlerFunc are startup helpers that panic on invalid inputs.

A handler coordinates local same-key callers with cancelable admission and uses a token-bearing Redis entry for cross-process admission. Completed results are reused while retained; different requests under the same key return ErrRequestMismatch. Cross-process calls observing an unfinished entry return ErrRequestInFlight.

Tasks execute synchronously and may run concurrently for different keys. Renewal loss cancels the task context and prevents publishing a successful response. Renewal is stopped and joined before replacing the in-flight entry with the completed result. Tasks must cooperate with cancellation and must not reenter the same key. Panics propagate after cleanup. Cleanup uses a five-second timeout detached from the caller and joins failures with operation errors.

This is a single-primary Redis lease and result cache, not an exactly-once guarantee for external effects. A task may run again after TTL expiry, failure, or data loss. Protect external effects through application idempotency or fencing when required. Request equality uses JSON serialization; use stable request representations. Clients require SETIFDEQ and DELEX support.

## Expected errors

`ErrEmptyKey` rejects an empty key. `ErrRequestMismatch` means the same key was used for another request. `ErrRequestInFlight` means another process is still working. `ErrLockConflict` means ownership was lost. Task and Redis errors propagate.

## Pitfalls

Retention has a TTL. Reuse a key only for the same logical request; protect external effects independently because lease loss or data loss can allow another execution.
