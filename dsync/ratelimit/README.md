# Redis rate limits

NewFixedWindow(client, Config{}) and NewGCRA(client, Config{}) return concrete limiters and errors. Config is passed by value and copied into private fields. Zero Limit and Period select 100 per minute; zero Burst disables extra burst. MustNew variants panic on invalid inputs. The Redis client is borrowed and caller-owned.

Run Setup on each Redis primary to install rl_fixed_window and rl_gcra. Deploy the updated functions with callers when migrating from v0.0.3 or earlier. Admission and result metadata are computed in one Redis function using the server clock. Empty keys and negative or nonrepresentable quantities are rejected. Zero quantity inspects without consuming capacity.

FixedWindow uses millisecond TTL precision. Rejected batches consume nothing. Burst is unused by that algorithm; bursts near window boundaries are inherent. GCRA admits a whole batch against Burst+1 instantaneous capacity, rejects oversized batches, and retains theoretical arrival time until debt is replenished. Emission intervals below one millisecond and overflowing burst durations are rejected during construction.

Remaining reports immediate capacity, RetryAfter applies to the requested batch, and ResetAfter reports the time until full replenishment. An impossible batch has no finite RetryAfter. Redis failures propagate; no background work is started. Atomicity covers one Redis primary and does not guarantee durability through data loss or failover.

## Migration from v0.0.x

Replace positional limit/period/burst parameters with Config values, and handle constructor errors. Reinstall functions when required above.
