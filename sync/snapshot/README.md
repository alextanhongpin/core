# Snapshot notifications

`New(Config{})` returns `(*Snapshot, stop, error)` and uses DefaultPolicies with an unbuffered input. MustNew panics on invalid configuration for startup wiring. Configuration is passed by value, policy slices are cloned and sorted privately, and later caller mutations do not affect behavior.

Nil Policies selects defaults; an explicit empty slice is invalid. Policy Changes must be positive and After nonnegative. Zero After allows triggering as soon as enough changes arrive. Zero BufferSize is unbuffered; negative buffer sizes return errors. Validate does not mutate configuration; WithDefaults returns a defaulted copy.

Add and Inc record changes. A notification is triggered when a policy's elapsed-time and change-count requirements both hold. Policies are checked in ascending After order. Notification resets the accumulated count and elapsed-time origin. Time checks use the smallest nonzero policy duration as their tick interval.

Subscribe with Chan or Go before expecting notifications. Broadcast acceptance does not confirm delivery. Slow subscribers apply backpressure to notifications and subsequent changes. Stop is idempotent and concurrent-safe, interrupts pending delivery, and waits for callbacks. Callbacks must finish and must not call stop or blocking operations on this snapshot. Chan after stop returns nil. Add after stop returns without work. Accepted notifications may be discarded during shutdown.
