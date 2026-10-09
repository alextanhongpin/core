# Redis rate limits

Share per-key fixed-window or GCRA quotas across service instances through Redis.

## Installation and documentation

```sh
go get github.com/alextanhongpin/core/dsync/ratelimit
go doc github.com/alextanhongpin/core/dsync/ratelimit
```

[Go API reference](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/ratelimit) · [Module requirements](go.mod)

## Typical use

Use for API quotas enforced by multiple replicas. Run Setup on each Redis primary before serving requests.

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
if err := ratelimit.Setup(ctx, client); err != nil {
	return err
}
limiter, err := ratelimit.NewFixedWindow(client, ratelimit.Config{
	Limit: 100, Period: time.Minute,
})
if err != nil {
	return err
}
result, err := limiter.Limit(ctx, "user:123")
if err != nil {
	return err
} // Backend or input failure.
if !result.Allow {
	fmt.Println("quota reached; retry after", result.RetryAfter)
	return nil
}
return handleRequest()
```

## Behavior and configuration

NewFixedWindow(client, Config{}) and NewGCRA(client, Config{}) return concrete limiters and errors. Config is passed by value and copied into private fields. Zero Limit and Period select 100 per minute; zero Burst disables extra burst. MustNew variants panic on invalid inputs. The Redis client is borrowed and caller-owned.

Run Setup on each Redis primary to install rl_fixed_window and rl_gcra. Admission and result metadata are computed in one Redis function using the server clock. Empty keys and negative or nonrepresentable quantities are rejected. Zero quantity inspects without consuming capacity.

FixedWindow uses millisecond TTL precision. Rejected batches consume nothing. Burst is unused by that algorithm; bursts near window boundaries are inherent. GCRA admits a whole batch against Burst+1 instantaneous capacity, rejects oversized batches, and retains theoretical arrival time until debt is replenished. Emission intervals below one millisecond and overflowing burst durations are rejected during construction.

Remaining reports immediate capacity, RetryAfter applies to the requested batch, and ResetAfter reports the time until full replenishment. An impossible batch has no finite RetryAfter. Redis failures propagate; no background work is started. Atomicity covers one Redis primary and does not guarantee durability through data loss or failover.

## Expected errors

`Allow` returns `(false, nil)` for a quota denial. `ErrNegative` rejects negative quantities. Invalid keys, invalid configuration, unsupported Redis commands, and backend failures return errors.

## Pitfalls

Handle a quota denial separately from Redis failure. GCRA requires millisecond emission intervals; fixed windows can admit bursts near boundaries.
