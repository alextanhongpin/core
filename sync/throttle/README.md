# Concurrency throttle

`New(Config{}) (*Throttler, error)` uses a default concurrency limit of 1000, no backlog, and no admission waiting. `DefaultConfig()` explicitly enables 100 queued calls and a ten-second admission timeout. `MustNew` panics on invalid configuration for startup wiring.

Configuration is passed by value and stored privately. WithDefaults preserves zero backlog and timeout. Validate rejects negative settings, nonpositive effective limits, and total-capacity overflow.

Do executes the callback synchronously after admission, or rejects without calling it. It returns ErrCapacityExceeded when all running and backlog slots are occupied. BacklogTimeout limits admission waiting only; the callback receives the original caller context. Callbacks may execute concurrently and must cooperate with cancellation. Admission order is not FIFO. Permits are returned even if the callback panics.

Func preserves the context/request/result signature. Outside retry, one permit covers attempts and backoff; inside retry, each attempt obtains its own permit.
