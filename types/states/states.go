// Package states validates boolean conditions and workflow state.
//
// It has three parts:
//
//   - counting helpers (ExactlyOne, AtLeastN, AllOrNone, Majority, ...) for
//     validating combinations of optional fields or flags;
//   - Sequence, which checks that ordered steps were completed in order;
//   - StateMachine, which allows only declared transitions between states.
package states

import (
	"fmt"
	"strings"
)

// Status is the state of a Sequence.
type Status int

const (
	// Idle means no step is complete.
	Idle Status = iota
	// Pending means some, but not all, leading steps are complete.
	Pending
	// Success means every step is complete.
	Success
	// Failed means a step is complete while an earlier one is not.
	Failed
)

// String returns the lowercase name of the status.
func (s Status) String() string {
	switch s {
	case Idle:
		return "idle"
	case Pending:
		return "pending"
	case Success:
		return "success"
	case Failed:
		return "failed"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// IsComplete reports whether the status is final: Success or Failed.
func (s Status) IsComplete() bool {
	return s == Success || s == Failed
}

// Step is a named condition in a Sequence.
type Step struct {
	name string
	cond func() bool
}

// NewStep returns a step whose condition is fixed.
func NewStep(name string, done bool) Step {
	return Step{name: name, cond: func() bool { return done }}
}

// NewStepFunc returns a step whose condition is evaluated on every check.
func NewStepFunc(name string, cond func() bool) Step {
	return Step{name: name, cond: cond}
}

// Name returns the step's name.
func (s Step) Name() string { return s.name }

// Check evaluates the step's condition. A step with no condition is not done.
func (s Step) Check() bool { return s.cond != nil && s.cond() }

// Sequence checks ordered steps, such as "email verified, then profile
// filled, then payment added". Steps must be completed from the start with no
// gaps; a later step done before an earlier one makes the sequence Failed.
//
// Every method evaluates each step's condition exactly once, so a result is
// consistent even when conditions change between calls.
type Sequence struct {
	steps []Step
}

// NewSequence returns a sequence of the given steps in order.
func NewSequence(steps ...Step) *Sequence {
	return &Sequence{steps: append([]Step(nil), steps...)}
}

type snapshot struct {
	done      []bool
	completed int // number of true entries
	next      int // index of the first false entry, or len(done)
}

func (s *Sequence) eval() snapshot {
	snap := snapshot{done: make([]bool, len(s.steps)), next: -1}
	for i, step := range s.steps {
		ok := step.Check()
		snap.done[i] = ok
		if ok {
			snap.completed++
		} else if snap.next < 0 {
			snap.next = i
		}
	}
	if snap.next < 0 {
		snap.next = len(s.steps)
	}
	return snap
}

func (snap snapshot) valid() bool { return snap.completed == snap.next }

func (snap snapshot) status() Status {
	switch {
	case !snap.valid():
		return Failed
	case snap.next == len(snap.done):
		return Success
	case snap.next == 0:
		return Idle
	default:
		return Pending
	}
}

// Status returns Success when all steps are done (including for an empty
// sequence), Idle when none are, Pending when a leading run is done, and
// Failed when steps were done out of order.
func (s *Sequence) Status() Status { return s.eval().status() }

// IsValid reports whether completed steps form an unbroken run from the
// first step, that is, whether Status is not Failed.
func (s *Sequence) IsValid() bool { return s.eval().valid() }

// Next returns the first step that is not done. It returns false when every
// step is done or the sequence is Failed.
func (s *Sequence) Next() (Step, bool) {
	snap := s.eval()
	if !snap.valid() || snap.next == len(s.steps) {
		return Step{}, false
	}
	return s.steps[snap.next], true
}

// CompletedSteps returns how many steps are done, in any order.
func (s *Sequence) CompletedSteps() int { return s.eval().completed }

// TotalSteps returns the number of steps.
func (s *Sequence) TotalSteps() int { return len(s.steps) }

// Steps returns a copy of the steps.
func (s *Sequence) Steps() []Step { return append([]Step(nil), s.steps...) }

// String describes the status and each step, for example
// "pending [x] email [ ] profile".
func (s *Sequence) String() string {
	snap := s.eval()
	var b strings.Builder
	b.WriteString(snap.status().String())
	for i, step := range s.steps {
		if snap.done[i] {
			b.WriteString(" [x] ")
		} else {
			b.WriteString(" [ ] ")
		}
		b.WriteString(step.name)
	}
	return b.String()
}
