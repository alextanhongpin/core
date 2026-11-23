# backoff

Delay policies for retries, polling, and reconnect loops. A policy maps an
attempt index to a `time.Duration`; it does not sleep, run your operation, or
look at a context.

**Use it for:** choosing how long to wait between attempts. It is the delay
source for [`sync/retry`](../retry), and works equally in a hand-written loop.

**Don't use it for:** executing or cancelling work, retry budgets, or error
classification (use [`sync/retry`](../retry)).

## Installation

```sh
go get github.com/alextanhongpin/core/sync/backoff
```

Check the module's `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/backoff"
)

func main() {
	policy := backoff.NewExponential(10*time.Millisecond, 100*time.Millisecond)
	ctx := context.Background()

	calls := 0
	op := func() error {
		calls++
		if calls < 3 {
			return errors.New("temporary")
		}
		return nil
	}

	for attempt := 1; ; attempt++ {
		if err := op(); err == nil {
			break
		}
		timer := time.NewTimer(policy.At(attempt)) // you own the waiting
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
	fmt.Println("calls:", calls)
}
```

It prints `calls: 3`.

## API

```go
type Backoff interface{ At(attempts int) time.Duration }

func NewConstant(period time.Duration) *Constant      // always period
func NewLinear(period time.Duration) *Linear          // period * index
func NewExponential(base, cap time.Duration) *Exponential // full jitter below min(base * 2^index, cap)

type Constant struct{ Period time.Duration }
type Linear struct{ Period time.Duration }
type Exponential struct{ Base, Cap time.Duration }
```

## Behavior and guarantees

- `sync/retry` calls `At(1)` for its first retry, so exponential backoff's first
  delay is below `2 * base`. Indices are chosen by the caller.
- `Constant`: nonpositive period gives zero.
- `Linear`: `period * index`, saturating at the maximum `time.Duration`.
  Nonpositive period or index gives zero.
- `Exponential`: a uniformly random delay in `[0, min(base * 2^index, cap))`
  (full jitter), sampled anew on every call, so it can be much shorter than the
  bound. Negative indices use 0 and indices above 62 use 62. Nonpositive `base`
  or `cap` gives zero. The bound is computed in floating point and compared with
  `cap` before conversion to `Duration`, so overflowing values saturate at `cap`.
- Zero-value structs return zero delay. Exported fields may be set directly;
  do not mutate them concurrently with calls to `At`. Otherwise all policies are
  safe for concurrent use.
- Nothing here waits or cancels: use a context-aware timer, as above.

## Testing

```sh
go vet ./...
go test -race -count=1 ./...
```
