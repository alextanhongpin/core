# Redis-backed idempotent tasks

New(client) and NewWithRedis(redisClient) return an Idempotent and error. Clients remain borrowed. HandlerFunc(fn, HandlerConfig{}) and Handler(task, HandlerConfig{}) return a handler and error. Configuration is passed by value and captured privately. Zero LockTTL and KeepTTL select ten seconds and 24 hours; TTLs below one millisecond are rejected. WithDefaults returns a copy and Validate does not mutate it. MustNew, MustNewWithRedis, MustHandler, and MustHandlerFunc are startup helpers that panic on invalid inputs.

A handler coordinates local same-key callers with cancelable admission and uses a token-bearing Redis entry for cross-process admission. Completed results are reused while retained; different requests under the same key return ErrRequestMismatch. Cross-process calls observing an unfinished entry return ErrRequestInFlight.

Tasks execute synchronously and may run concurrently for different keys. Renewal loss cancels the task context and prevents publishing a successful response. Renewal is stopped and joined before replacing the in-flight entry with the completed result. Tasks must cooperate with cancellation and must not reenter the same key. Panics propagate after cleanup. Cleanup uses a five-second timeout detached from the caller and joins failures with operation errors.

This is a single-primary Redis lease and result cache, not an exactly-once guarantee for external effects. A task may run again after TTL expiry, failure, or data loss. Protect external effects through application idempotency or fencing when required. Request equality uses JSON serialization; use stable request representations. Clients require SETIFDEQ and DELEX support.

## Migration from v0.0.x

Pass HandlerConfig values instead of pointers/nil and handle construction errors. DefaultConfig returns a value. Use Must variants only for startup wiring.
