# snapshot

Notify subscribers when enough changes have accumulated and enough time has
passed, with stricter time requirements for smaller change counts.

**Use it for:** triggering periodic persistence of in-memory state (write a
snapshot after 1,000 changes and 1s, or after 1 change and 1h), flushing more
often as write volume grows.

**Don't use it for:** durable job queues (notifications are triggers, not
jobs), or plain fan-out of events (use [`sync/broadcast`](../broadcast), which
snapshot builds on).

## Installation

```sh
go get github.com/alextanhongpin/core/sync/snapshot
```

## Quick start

```go
package main

import (
	"fmt"
	"log"

	"github.com/alextanhongpin/core/sync/snapshot"
)

func main() {
	s, stop, err := snapshot.New(snapshot.Config{
		Policies: []snapshot.Policy{{Changes: 10}},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stop()

	notifications := s.Chan() // subscribe before recording changes
	s.Add(10)
	p := <-notifications
	fmt.Println("persist after", p.Changes, "changes")
}
```

It prints `persist after 10 changes`.

## API

```go
type Policy struct {
	After   time.Duration // minimum time since the last notification
	Changes int           // minimum accumulated changes
}

type Config struct {
	BufferSize int      // admission buffer for Add; 0 is unbuffered
	Policies   []Policy // nil selects DefaultPolicies
}

func DefaultPolicies() []Policy
func DefaultConfig() Config
func (Config) WithDefaults() Config // fills nil Policies
func (Config) Validate() error

func New(cfg Config) (*Snapshot, func(), error) // snapshot and idempotent stop
func MustNew(cfg Config) (*Snapshot, func())    // panics on invalid cfg

func (*Snapshot) Add(n int) // record n changes; n <= 0 is ignored
func (*Snapshot) Inc()      // Add(1)

// Promoted from the embedded *broadcast.Broadcast[Policy]:
func (*Snapshot) Chan() <-chan Policy
func (*Snapshot) Go(fn func(Policy))
func (*Snapshot) Send(p Policy) // exposed, but not meant for callers
```

## Behavior and guarantees

- **Firing rule:** policies are sorted (stably) by `After`. On each `Add` and
  each tick, policies are checked in ascending `After` order; the first one
  whose `After` has elapsed since the last notification and whose `Changes`
  threshold is met fires. Firing resets the count and the time origin, and
  restarts the tick cadence, so a time-based notification is due about one
  smallest-`After` interval later.
- **Ticks:** time checks run at the smallest nonzero `After`. With `After: 0`
  a policy fires as soon as enough changes arrive.
- **Counter:** `Add` ignores `n <= 0` and saturates at the largest `int`.
- **Validation:** `Changes` must be positive, `After` nonnegative,
  `BufferSize` nonnegative, and an explicit empty `Policies` is invalid (nil
  means defaults). `New` clones and sorts the policies; later caller mutation
  has no effect.
- **Subscribers:** subscribe with `Chan` or `Go` before expecting
  notifications. The embedded broadcast is exposed, so `Send`, `Chan` and `Go`
  are callable on `*Snapshot`; do not call `Send` yourself. Slow subscribers
  apply backpressure to notifications and to later `Add` calls.
- **Stop:** idempotent and concurrent-safe; interrupts pending delivery and
  waits for `Go` callbacks. Callbacks must finish and must not call stop or
  blocking operations on the snapshot. `Add` after stop does nothing.
  `Chan` after stop returns a closed channel.
- Notifications may be dropped during shutdown; they are triggers, not durable.

## Testing

```sh
go vet ./...
go test -race -count=1 ./...
go test -count=1 ./...
```

Timing tests run under `testing/synctest`, so they use a fake clock.
