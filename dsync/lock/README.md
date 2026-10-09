# Redis leases

New(client, Config{}) returns a Locker and error. Configuration is passed by value, defaulted, validated, and stored privately. Zero LockTTL selects 30 seconds; zero RefreshRatio disables renewal. DefaultConfig explicitly enables renewal at ratio 0.8. Nil Retry selects a contention-only retry policy; nil Logger selects slog.Default. MustNew is the panicking startup helper.

Do admits same-instance callers by key with cancelable local admission, acquires a unique Redis token, and invokes the callback synchronously. On caller cancellation or lease loss it cancels the callback context and waits for callback completion before cleanup. Callbacks must cooperate with cancellation and must not reenter the same key. Panics propagate after cleanup; cleanup errors are joined with operation errors. Cleanup has a five-second timeout detached from caller cancellation.

Callbacks, retry runners, and Redis clients remain shared dependencies and must support concurrent use. Func returns the final callback value and error; resource-bearing callback results remain caller-owned. No instance owns the Redis client's lifetime.

NewClient uses token-checked SET NX acquisition and SETIFDEQ/DELEX renewal/release, requiring a Redis release that supports these commands. These are single-primary leases, not fencing: expiration, failover, or paused processes can permit overlapping external work. Use fencing at the protected resource when required.
