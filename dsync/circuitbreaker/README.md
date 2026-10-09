# Redis circuit breaker

New(client, Config{}) returns a breaker and error. MustNew is the panicking startup helper. Configuration is passed by value and stored privately; the borrowed client and weighting callbacks remain shared and must support concurrent calls. Zero thresholds and durations select defaults. Durations must be at least one millisecond. Nil hooks select defaults; supply zero-returning hooks to disable extra weighting.

Run Setup on every Redis primary before use. It installs cb_begin, cb_commit, and cb_set_status functions. Deploy callers and functions together; Setup replaces the function library. Redis must support functions and hash-field expiration.

Do admits once, executes the callback synchronously, and rejects opened states with ErrOpened. Callbacks run without a local mutex and may execute concurrently. Failed calls contribute 1+FailureCount(err)+SlowCallCount(duration); successful slow calls do not count as failures. Weight hooks must be nonnegative. Half-open closes at the exact configured success threshold. It does not limit concurrent probes.

State transitions advance a Redis generation. Results admitted in an earlier generation cannot change a newer state. SetStatus(Opened) starts its timeout. Failed-operation commits use a five-second context detached from caller cancellation, preserving both operation and Redis errors. State keys remain until externally removed; deleting a key destroys generation history. Single-primary state does not promise continuity through Redis failover or data loss.

Func preserves the operation signature and caller context. With retry outside the breaker, each attempt contributes separately; outside retry, the breaker observes the logical operation once.
