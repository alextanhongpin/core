# Circuit breaker

Stop repeatedly calling a failing dependency and allow recovery probes.

## Installation and documentation

```sh
go get github.com/alextanhongpin/core/sync/circuitbreaker
go doc github.com/alextanhongpin/core/sync/circuitbreaker
```

[Go API reference](https://pkg.go.dev/github.com/alextanhongpin/core/sync/circuitbreaker) · [Module requirements](go.mod)

## Typical use

Use around an unreliable downstream API or database operation. Share a breaker across calls to the same dependency.

The usage example below shows the main operation. Application types and
callbacks such as `User` and `fetchUsers` belong to your application; import this
package and the standard packages referenced in the snippet.

## Behavior and configuration

Construct with `New(Config{}) (*CircuitBreaker, error)` for defaults, or `MustNew(Config{})` for startup wiring that panics on invalid configuration. Configuration is passed by value and stored privately. Zero thresholds and durations select defaults; negative values are rejected. Nil weighting hooks select defaults; supply zero-returning hooks to disable extra weighting.

```go
cfg := circuitbreaker.DefaultConfig()
cfg.FailureThreshold = 3
cb, err := circuitbreaker.New(cfg)
if err != nil { return err }
return cb.Do(func() error { return operation() })
```

Instances support concurrent operations. Hooks and operations may run concurrently and remain caller-owned. Callbacks run outside the state lock. Each call invokes its operation once or rejects with ErrOpened. Results admitted in an earlier state cannot change a newer state.

Failure weighting is `1 + FailureCount(err) + SlowCallCount(duration)` for failed calls only. Successful slow calls do not add failures. Hooks must return nonnegative weights. The counter expires after an idle FailurePeriod or SuccessPeriod. Half-open does not impose a concurrency limit; combine with throttle when probes need bounded concurrency.

`Func` preserves the operation signature and caller context. With retry outside the breaker, each attempt is counted separately; with the breaker outside retry, the entire logical operation is counted once. `NewTransporter` defaults a nil transport to http.DefaultTransport, classifies 5xx as failures, and closes discarded response bodies. The breaker dependency must be non-nil.

## Expected errors

`ErrOpened` means the operation was rejected without running. Otherwise `Do` returns the operation error. Invalid configuration fails at construction.

## Pitfalls

Half-open probes are not concurrency-limited. Combine with a throttle when a recovering dependency needs a small number of probes.
