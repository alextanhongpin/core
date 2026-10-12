# backoff

[Go reference](https://pkg.go.dev/github.com/alextanhongpin/core/sync/backoff)

Calculate delays for retries, polling, and reconnect loops. This package does not wait, execute operations, or manage cancellation. Its Go version is specified in `go.mod`.

```sh
go get github.com/alextanhongpin/core/sync/backoff
```

```go
import (
    "time"
    "github.com/alextanhongpin/core/sync/backoff"
)

policy := backoff.NewExponential(100*time.Millisecond, 2*time.Second)
delay := policy.At(1) // Full jitter: 0 <= delay < 200ms.
```

- `NewConstant(period)` returns the same delay for every index; nonpositive periods return zero.
- `NewLinear(period)` returns `period * index`, saturated at the maximum duration. Nonpositive periods or indices return zero.
- `NewExponential(base, cap)` returns full jitter in `[0, min(base * 2^index, cap))`, with overflow capped. Negative indices use zero; indices above 62 use 62. Nonpositive base or cap returns zero.

Indices are supplied by the caller. The retry package uses index 1 for its first retry, giving exponential backoff an initial upper bound of twice the base.

The strategies implement `Backoff`, which requires `At(int) time.Duration`. Exported fields support struct literals and adjustment before use. Zero-valued strategies return zero. Concurrent calls are safe when fields are not mutated concurrently. Exponential calls sample a new random delay each time.

Use a context-aware timer when waiting. The caller decides when to stop and whether an operation is safe to repeat. See the runnable example in `backoff_test.go`.
