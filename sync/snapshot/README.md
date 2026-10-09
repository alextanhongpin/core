# Snapshot notifications

Notify subscribers when both a change count and elapsed-time threshold are reached.

## Installation and documentation

```sh
go get github.com/alextanhongpin/core/sync/snapshot
go doc github.com/alextanhongpin/core/sync/snapshot
```

[Go API reference](https://pkg.go.dev/github.com/alextanhongpin/core/sync/snapshot) · [Module requirements](go.mod)

## Typical use

Use to trigger periodic persistence of in-memory state, flushing more frequently as writes increase.

## Quick start

The snippet belongs inside a function returning `error`; application functions
such as `handleRequest` represent your own work. Import the package above and
the standard packages used in the snippet (`context`, `errors`, `fmt`, or `time`).

```go
s, stop, err := snapshot.New(snapshot.Config{
	Policies: []snapshot.Policy{{Changes: 10, After: time.Second}},
})
if err != nil {
	return err
}
defer stop()
notifications := s.Chan() // Subscribe before recording changes.
s.Add(10)
select {
case policy := <-notifications:
	fmt.Println("persist changes after", policy.After)
case <-ctx.Done():
	return ctx.Err()
}
```

## Behavior and configuration

`New(Config{})` returns `(*Snapshot, stop, error)` and uses DefaultPolicies with an unbuffered input. MustNew panics on invalid configuration for startup wiring. Configuration is passed by value, policy slices are cloned and sorted privately, and later caller mutations do not affect behavior.

Nil Policies selects defaults; an explicit empty slice is invalid. Policy Changes must be positive and After nonnegative. Zero After allows triggering as soon as enough changes arrive. Zero BufferSize is unbuffered; negative buffer sizes return errors. Validate does not mutate configuration; WithDefaults returns a defaulted copy.

Add and Inc record changes. A notification is triggered when a policy's elapsed-time and change-count requirements both hold. Policies are checked in ascending After order. Notification resets the accumulated count and elapsed-time origin. Time checks use the smallest nonzero policy duration as their tick interval.

Subscribe with Chan or Go before expecting notifications. Broadcast acceptance does not confirm delivery. Slow subscribers apply backpressure to notifications and subsequent changes. Stop is idempotent and concurrent-safe, interrupts pending delivery, and waits for callbacks. Callbacks must finish and must not call stop or blocking operations on this snapshot. Chan after stop returns nil. Add after stop returns without work. Accepted notifications may be discarded during shutdown.

## Expected errors

Construction rejects empty policies, nonpositive change thresholds, negative durations, and negative buffers. `Add` and `Inc` have no error return; calls after shutdown do nothing.

## Pitfalls

Notifications are triggers, not durable jobs. Slow subscribers block progress, and callbacks must finish before shutdown can complete.
