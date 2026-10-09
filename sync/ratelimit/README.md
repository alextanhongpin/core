# Per-key rate limiting

Limit requests per key within one process using fixed-window or GCRA admission.

## Installation and documentation

```sh
go get github.com/alextanhongpin/core/sync/ratelimit
go doc github.com/alextanhongpin/core/sync/ratelimit
```

[Go API reference](https://pkg.go.dev/github.com/alextanhongpin/core/sync/ratelimit) · [Module requirements](go.mod)

## Typical use

Use for per-user API quotas in a single service instance. Choose GCRA for smoother admission and FixedWindow for simple window quotas.

## Quick start

The snippet belongs inside a function returning `error`; application functions
such as `handleRequest` represent your own work. Import the package above and
the standard packages used in the snippet (`context`, `errors`, `fmt`, or `time`).

```go
limiter, err := ratelimit.NewGCRA(ratelimit.Config{
	Limit: 100, Period: time.Minute, Burst: 5,
})
if err != nil {
	return err
}
result := limiter.Limit("user:123")
if !result.Allow {
	fmt.Println("try again after", result.RetryAfter)
	return nil // Denied: skip the protected operation.
}
return handleRequest()
```

## Behavior and configuration

NewFixedWindow(Config{}) and NewGCRA(Config{}) return a concrete limiter and error. Configuration is copied into private operational fields. Zero limit and period select 100 admissions per minute; zero burst means no extra burst allowance. DefaultConfig returns a value. WithDefaults returns an effective copy and Validate checks it without mutation. MustNewFixedWindow and MustNewGCRA are startup helpers that panic on invalid configuration.

Both limiters support concurrent calls. Empty keys and negative quantities are rejected. Zero quantity inspects allowance without consuming tokens. FixedWindow starts a window at the first admission for a key and resets it after Period. Rejected batches do not consume tokens. Burst is unused by FixedWindow; boundary bursts are inherent to fixed windows.

GCRA spaces emissions by Period/Limit and admits batches atomically against Burst+1 instantaneous capacity. Oversized batches are rejected. Construction rejects sub-nanosecond intervals and overflowing burst allowance. Remaining reflects immediate capacity; Limit reports the configured rate. RetryAfter is the delay needed for the requested batch; GCRA ResetAfter currently equals RetryAfter, rather than the time to fully replenish burst capacity.

State is retained per key until Clear removes expired entries. Call Clear periodically when cardinality is unbounded; neither limiter starts a cleanup goroutine.

Func obtains a key and admits once before invoking its operation. HTTP emits Retry-After rounded up to whole seconds when rejected. Key functions and downstream operations are caller-owned and may execute concurrently. With retry outside the decorator, each attempt consumes admission; inside, one admission covers the logical operation.

## Expected errors

`Allow` returns false for denied or invalid requests; it has no error result. `Limit` returns metadata with `Allow=false`. The `Func` decorator returns `ErrTooManyRequests` on rejection; HTTP responds with status 429.

## Pitfalls

Instances do not share quotas across processes. Clear expired state periodically when keys are unbounded.
