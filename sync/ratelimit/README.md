# Per-key rate limiting

NewFixedWindow(Config{}) and NewGCRA(Config{}) return a concrete limiter and error. Configuration is copied into private operational fields. Zero limit and period select 100 admissions per minute; zero burst means no extra burst allowance. DefaultConfig returns a value. WithDefaults returns an effective copy and Validate checks it without mutation. MustNewFixedWindow and MustNewGCRA are startup helpers that panic on invalid configuration.

Both limiters support concurrent calls. Empty keys and negative quantities are rejected. Zero quantity inspects allowance without consuming tokens. FixedWindow starts a window at the first admission for a key and resets it after Period. Rejected batches do not consume tokens. Burst is unused by FixedWindow; boundary bursts are inherent to fixed windows.

GCRA spaces emissions by Period/Limit and admits batches atomically against Burst+1 instantaneous capacity. Oversized batches are rejected. Construction rejects sub-nanosecond intervals and overflowing burst allowance. Remaining reflects immediate capacity; Limit reports the configured rate. RetryAfter is the delay needed for the requested batch; GCRA ResetAfter currently equals RetryAfter, rather than the time to fully replenish burst capacity.

State is retained per key until Clear removes expired entries. Call Clear periodically when cardinality is unbounded; neither limiter starts a cleanup goroutine.

Func obtains a key and admits once before invoking its operation. HTTP emits Retry-After rounded up to whole seconds when rejected. Key functions and downstream operations are caller-owned and may execute concurrently. With retry outside the decorator, each attempt consumes admission; inside, one admission covers the logical operation.
