// Package states provides utilities for managing sequential state validation
// and state machine transitions. It ensures all required steps are completed
// before proceeding to the next step in a workflow.
package states

import "fmt"

// Handler represents a condition function that returns true when satisfied.
type Handler func() bool

// Step represents a single step in a sequence with a name and condition.
type Step struct {
	name      string
	condition Handler
}

// Name returns the step's name.
func (s Step) Name() string {
	return s.name
}

// Check evaluates the step's condition.
func (s Step) Check() bool {
	return s.condition()
}

// NewStep creates a new step with a static boolean condition.
func NewStep(name string, satisfied bool) Step {
	return Step{
		name: name,
		condition: func() bool {
			return satisfied
		},
	}
}

// NewStepFunc creates a new step with a dynamic condition function.
func NewStepFunc(name string, condition Handler) Step {
	return Step{
		name:      name,
		condition: condition,
	}
}

// Sequence manages a series of steps that must be completed in order.
// Each step must be satisfied before the next step becomes available.
type Sequence struct {
	steps []Step
}

// NewSequence creates a new sequence with the given steps.
func NewSequence(steps ...Step) *Sequence {
	return &Sequence{
		steps: steps,
	}
}

// Status returns the current status of the sequence.
func (s *Sequence) Status() Status {
	if !s.IsValid() {
		return Failed
	}

	nextIndex := s.nextStepIndex()
	switch {
	case nextIndex == 0:
		return Idle
	case nextIndex == len(s.steps):
		return Success
	default:
		return Pending
	}
}

// Next returns the next step that needs to be completed and whether one exists.
// Returns false if the sequence is invalid or complete.
func (s *Sequence) Next() (Step, bool) {
	if !s.IsValid() {
		return Step{}, false
	}

	nextIndex := s.nextStepIndex()
	if nextIndex >= len(s.steps) {
		return Step{}, false
	}

	return s.steps[nextIndex], true
}

// IsValid checks if the sequence is in a valid state.
// A sequence is valid if all completed steps form a contiguous sequence from the beginning.
func (s *Sequence) IsValid() bool {
	progress := s.calculateProgress()
	nextIndex := s.nextStepIndex()

	// Valid if progress matches the expected cumulative pattern
	// For example: if next step is index 2, progress should be 0b11 (3)
	expectedProgress := (1 << nextIndex) - 1
	return progress == expectedProgress
}

// Progress returns the current progress as a bitmask.
func (s *Sequence) Progress() int {
	return s.calculateProgress()
}

// CompletedSteps returns the number of completed steps.
func (s *Sequence) CompletedSteps() int {
	count := 0
	for _, step := range s.steps {
		if step.Check() {
			count++
		}
	}
	return count
}

// TotalSteps returns the total number of steps in the sequence.
func (s *Sequence) TotalSteps() int {
	return len(s.steps)
}

// GetStep returns the step at the given index.
func (s *Sequence) GetStep(index int) (Step, bool) {
	if index < 0 || index >= len(s.steps) {
		return Step{}, false
	}
	return s.steps[index], true
}

// AllSteps returns all steps in the sequence.
func (s *Sequence) AllSteps() []Step {
	result := make([]Step, len(s.steps))
	copy(result, s.steps)
	return result
}

// nextStepIndex finds the index of the next incomplete step.
func (s *Sequence) nextStepIndex() int {
	for i, step := range s.steps {
		if !step.Check() {
			return i
		}
	}
	return len(s.steps)
}

// calculateProgress creates a bitmask representing which steps are complete.
func (s *Sequence) calculateProgress() int {
	var progress int
	for i, step := range s.steps {
		if step.Check() {
			progress |= 1 << i
		}
	}
	return progress
}

// String returns a string representation of the sequence status.
func (s *Sequence) String() string {
	status := s.Status()
	completed := s.CompletedSteps()
	total := s.TotalSteps()

	return fmt.Sprintf("Sequence[%s: %d/%d steps completed]",
		status.String(), completed, total)
}
