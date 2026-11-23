# states

Validate combinations of boolean conditions, ordered workflow steps, and state
transitions.

The package has three parts:

- **Counting helpers** (`ExactlyOne`, `ExactlyN`, `AtLeastN`, `AtMostN`,
  `AllOrNone`, `Majority`) for rules such as "exactly one of person or
  organization is set" or "start and end date are given together".
- **`Sequence`**, which checks that ordered steps were completed in order, such
  as onboarding or approval flows.
- **`StateMachine`**, which holds a current state and allows only declared,
  named transitions.

**Use it for:** validating domain invariants and in-memory workflow state.

**Don't use it for:** durable or distributed workflows (it keeps no history and
persists nothing), or plain any/all checks (use `slices.Contains(conds, true)`
or `!slices.Contains(conds, false)`).

## Installation

```sh
go get github.com/alextanhongpin/core/types
```

All `types` packages ship in the one `github.com/alextanhongpin/core/types`
module. Import this one as `github.com/alextanhongpin/core/types/states`. Check
the module's `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/types/states"
)

func main() {
	// Counting helpers.
	person, organization := true, false
	fmt.Println(states.ExactlyOne(person, organization))

	// Sequence.
	onboarding := states.NewSequence(
		states.NewStep("verify email", true),
		states.NewStep("complete profile", false),
		states.NewStep("add payment", false),
	)
	next, _ := onboarding.Next()
	fmt.Println(onboarding.Status(), next.Name())

	// StateMachine.
	sm := states.NewStateMachine("pending",
		states.NewTransition("pay", "pending", "paid"),
		states.NewTransition("cancel", "pending", "cancelled"),
		states.NewTransition("cancel", "paid", "cancelled"),
	)
	sm.AddListener(func(t states.Transition[string]) { fmt.Println(t) })
	_ = sm.Execute("pay")
	_ = sm.Execute("cancel")
	err := sm.Execute("pay")
	fmt.Println(sm.State(), errors.Is(err, states.ErrUnknownTransition))
}
```

It prints:

```
true
pending complete profile
pay: pending -> paid
cancel: paid -> cancelled
cancelled true
```

## API

```go
// Counting helpers.
func ExactlyN(n int, conditions ...bool) bool
func ExactlyOne(conditions ...bool) bool
func AtLeastN(n int, conditions ...bool) bool
func AtMostN(n int, conditions ...bool) bool
func AllOrNone(conditions ...bool) bool
func Majority(conditions ...bool) bool

// Sequence.
type Status int // Idle, Pending, Success, Failed
func (s Status) String() string
func (s Status) IsComplete() bool

type Step struct{ /* unexported */ }
func NewStep(name string, done bool) Step
func NewStepFunc(name string, cond func() bool) Step
func (s Step) Name() string
func (s Step) Check() bool

func NewSequence(steps ...Step) *Sequence
func (s *Sequence) Status() Status
func (s *Sequence) IsValid() bool
func (s *Sequence) Next() (Step, bool)
func (s *Sequence) CompletedSteps() int
func (s *Sequence) TotalSteps() int
func (s *Sequence) Steps() []Step
func (s *Sequence) String() string

// StateMachine.
var ErrUnknownTransition, ErrInvalidTransition error

type Transition[T comparable] struct{ Name string; From, To T }
func NewTransition[T comparable](name string, from, to T) Transition[T]
type Listener[T comparable] func(t Transition[T])

func NewStateMachine[T comparable](initial T, transitions ...Transition[T]) *StateMachine[T]
func (sm *StateMachine[T]) State() T
func (sm *StateMachine[T]) AddTransition(ts ...Transition[T])
func (sm *StateMachine[T]) AddListener(fn Listener[T])
func (sm *StateMachine[T]) Execute(name string) error
func (sm *StateMachine[T]) ExecuteFunc(name string, fn func() error) error
func (sm *StateMachine[T]) TransitionTo(target T) error
func (sm *StateMachine[T]) CanTransitionTo(target T) bool
func (sm *StateMachine[T]) Available() []Transition[T]
func (sm *StateMachine[T]) States() []T
func (sm *StateMachine[T]) HasCycle() bool
```

## Behavior and guarantees

### Counting helpers

- `ExactlyOne` means exactly one is true. It is not a parity XOR:
  `ExactlyOne(true, true, true)` is false.
- `AtLeastN` is true for `n <= 0`; `AtMostN` is false for `n < 0`.
- `AllOrNone()` with no conditions is true. `Majority` needs strictly more than
  half, so a tie or no conditions is false.
- All conditions are already evaluated by the caller; nothing short-circuits.

### Sequence

- Steps must be done as an unbroken run from the first step. `Status` is
  `Success` when every step is done (including an empty sequence), `Idle` when
  none are, `Pending` for a leading run, and `Failed` when a step is done while
  an earlier one is not.
- `Next` returns the first step that is not done, and false when the sequence is
  complete or `Failed`.
- Each method call evaluates every step condition exactly once, so a single
  result is consistent even if conditions change. Separate calls may disagree
  if conditions change between them.
- A `Step` without a condition (the zero value) is not done.
- `Sequence` adds no locking; it is as safe for concurrent use as its step
  conditions.

### StateMachine

- `Execute(name)` fires the transition with that name whose `From` is the
  current state, so one name (such as `"cancel"`) can be declared from several
  states. Otherwise it returns an error wrapping `ErrUnknownTransition`.
- `TransitionTo(target)` uses the first declared transition from the current
  state to `target`, or returns an error wrapping `ErrInvalidTransition`.
- A self-transition is allowed only if it is declared.
- `ExecuteFunc` checks the transition, runs `fn` without holding the lock, and
  changes state only if `fn` returns nil. If another goroutine changes the
  state while `fn` runs, the transition is looked up again from the new state
  and may fail with `ErrUnknownTransition`.
- Listeners run synchronously after the state changes, outside the lock, so
  they may call the machine. Listeners for concurrent transitions may run
  concurrently.
- `States` returns the current state first, then states in order of first
  appearance in the transitions. `HasCycle` counts declared self-transitions.
- All methods are safe for concurrent use.

## Testing

```sh
go vet ./states/...
go test -race -count=1 ./states/...
```

Run from the `types` directory.
