package states

import "fmt"

// Status represents the current state of a sequence or state machine.
type Status int

const (
	// Idle indicates the sequence has not started (no steps completed).
	Idle Status = iota

	// Pending indicates the sequence is in progress (some steps completed).
	Pending

	// Success indicates the sequence is complete (all steps completed).
	Success

	// Failed indicates the sequence is in an invalid state.
	Failed
)

// Valid returns true if the status is a recognized value.
func (s Status) Valid() bool {
	switch s {
	case Idle, Pending, Success, Failed:
		return true
	default:
		return false
	}
}

// String returns the string representation of the status.
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

// IsComplete returns true if the status indicates completion (success or failure).
func (s Status) IsComplete() bool {
	return s == Success || s == Failed
}

// IsActive returns true if the status indicates the sequence is active (pending).
func (s Status) IsActive() bool {
	return s == Pending
}

// IsSuccessful returns true if the status indicates successful completion.
func (s Status) IsSuccessful() bool {
	return s == Success
}

// IsFailed returns true if the status indicates failure.
func (s Status) IsFailed() bool {
	return s == Failed
}
