# Dataloader

`New(ctx, batchFn, Config{})` returns `(*DataLoader, stop, error)`. It copies, defaults, and validates configuration before starting work. Zero batch size and interval select 25 keys and 16 milliseconds; zero buffer size keeps admission unbuffered. Negative settings and nil batch functions return errors. MustNew returns the loader and stop function, panicking on invalid inputs for startup wiring.

```go
dl, stop, err := dataloader.New(ctx, fetchUsers, dataloader.Config{BatchSize: 50})
if err != nil { return err }
defer stop()
user, err := dl.LoadContext(ctx, userID)
```

The batch function receives the loader lifetime context and executes serially. It returns a map of results; missing keys return ErrNotFound. Concurrent callers with the same key share an in-flight future. Values are weakly cached and may be loaded again after garbage collection; this is not a permanent result cache. Returned mutable values remain shared.

Load blocks through admission and completion using the loader context. LoadMany returns results in input order with per-key errors; its outer error can report lifetime cancellation. LoadContext cancels an individual wait without canceling shared loading. Cancellation racing with completion may return either outcome. Func requires LoadContext and passes the caller context to loading and transformation.

Admission for new futures created by LoadContext is loader-owned. BufferSize does not bound the number of pending keys or callers. Callers should bound concurrency when key cardinality is unbounded.

Stop is concurrent-safe and idempotent. It cancels pending loads with ErrCanceled and waits for the worker and admission goroutines. The batch function must cooperate with cancellation for prompt shutdown and must not invoke stop itself. Loads after stop return the lifetime cancellation cause.

## Migration from v0.1.x

Pass configuration values instead of pointers/nil and handle the additional New error result. DefaultConfig returns a value. Public embedded configuration has been removed; configure before construction.
