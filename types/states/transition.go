package states

import (
	"fmt"
)

// Transition represents a state transition with a name and source/target states.
type Transition[T comparable] struct {
	Name string
	From T
	To   T
}

// NewTransition creates a new transition.
func NewTransition[T comparable](name string, from, to T) Transition[T] {
	return Transition[T]{
		Name: name,
		From: from,
		To:   to,
	}
}

// String returns a string representation of the transition.
func (t Transition[T]) String() string {
	return fmt.Sprintf("%s: %v -> %v", t.Name, t.From, t.To)
}

// StateMachine manages state transitions with validation.
type StateMachine[T comparable] struct {
	currentState T
	transitions  []Transition[T]
	listeners    []StateChangeListener[T]
}

// StateChangeListener is called when a state transition occurs.
type StateChangeListener[T comparable] func(from, to T, transitionName string)

// NewStateMachine creates a new state machine with the given initial state and transitions.
func NewStateMachine[T comparable](initialState T, transitions ...Transition[T]) *StateMachine[T] {
	return &StateMachine[T]{
		currentState: initialState,
		transitions:  transitions,
		listeners:    make([]StateChangeListener[T], 0),
	}
}

// State returns the current state.
func (sm *StateMachine[T]) State() T {
	return sm.currentState
}

// AddTransition adds a new transition to the state machine.
func (sm *StateMachine[T]) AddTransition(transition Transition[T]) {
	sm.transitions = append(sm.transitions, transition)
}

// AddListener adds a state change listener.
func (sm *StateMachine[T]) AddListener(listener StateChangeListener[T]) {
	sm.listeners = append(sm.listeners, listener)
}

// Execute performs a named transition if it's valid from the current state.
// Panics if the transition name is not found.
func (sm *StateMachine[T]) Execute(transitionName string) error {
	transition, found := sm.findTransitionByName(transitionName)
	if !found {
		return fmt.Errorf("transition %q not found", transitionName)
	}

	return sm.TransitionTo(transition.To, transitionName)
}

// TransitionTo attempts to transition to the target state.
// Returns an error if the transition is not valid.
func (sm *StateMachine[T]) TransitionTo(targetState T, transitionName string) error {
	if !sm.CanTransitionTo(targetState) {
		return fmt.Errorf("invalid transition from %v to %v", sm.currentState, targetState)
	}

	oldState := sm.currentState
	sm.currentState = targetState

	// Notify listeners
	for _, listener := range sm.listeners {
		listener(oldState, targetState, transitionName)
	}

	return nil
}

// TransitionWithFunc performs a transition with a function that can fail.
// If the function returns an error, the transition is not performed.
func (sm *StateMachine[T]) TransitionWithFunc(targetState T, transitionName string, fn func() error) error {
	if !sm.CanTransitionTo(targetState) {
		return fmt.Errorf("invalid transition from %v to %v", sm.currentState, targetState)
	}

	if err := fn(); err != nil {
		return err
	}

	return sm.TransitionTo(targetState, transitionName)
}

// CanTransitionTo checks if a transition to the target state is valid.
func (sm *StateMachine[T]) CanTransitionTo(targetState T) bool {
	if sm.currentState == targetState {
		return false // No self-transitions unless explicitly defined
	}

	return sm.IsValidTransition(sm.currentState, targetState)
}

// IsValidTransition checks if a transition between two states is defined.
func (sm *StateMachine[T]) IsValidTransition(from, to T) bool {
	for _, transition := range sm.transitions {
		if transition.From == from && transition.To == to {
			return true
		}
	}
	return false
}

// GetValidTransitions returns all valid transitions from the current state.
func (sm *StateMachine[T]) GetValidTransitions() []Transition[T] {
	var valid []Transition[T]
	for _, transition := range sm.transitions {
		if transition.From == sm.currentState {
			valid = append(valid, transition)
		}
	}
	return valid
}

// GetValidStates returns all states that can be transitioned to from the current state.
func (sm *StateMachine[T]) GetValidStates() []T {
	var states []T
	for _, transition := range sm.transitions {
		if transition.From == sm.currentState {
			states = append(states, transition.To)
		}
	}
	return states
}

// GetAllStates returns all unique states in the state machine.
func (sm *StateMachine[T]) GetAllStates() []T {
	stateSet := make(map[T]bool)

	// Add current state
	stateSet[sm.currentState] = true

	// Add all states from transitions
	for _, transition := range sm.transitions {
		stateSet[transition.From] = true
		stateSet[transition.To] = true
	}

	states := make([]T, 0, len(stateSet))
	for state := range stateSet {
		states = append(states, state)
	}

	return states
}

// GetTransitionByName finds a transition by name from the current state.
func (sm *StateMachine[T]) GetTransitionByName(name string) (Transition[T], bool) {
	for _, transition := range sm.transitions {
		if transition.Name == name && transition.From == sm.currentState {
			return transition, true
		}
	}
	var zero Transition[T]
	return zero, false
}

// Clone creates a copy of the state machine with the same configuration but reset to initial state.
func (sm *StateMachine[T]) Clone(initialState T) *StateMachine[T] {
	transitions := make([]Transition[T], len(sm.transitions))
	copy(transitions, sm.transitions)

	return &StateMachine[T]{
		currentState: initialState,
		transitions:  transitions,
		listeners:    make([]StateChangeListener[T], 0),
	}
}

// findTransitionByName finds any transition with the given name.
func (sm *StateMachine[T]) findTransitionByName(name string) (Transition[T], bool) {
	for _, transition := range sm.transitions {
		if transition.Name == name {
			return transition, true
		}
	}
	var zero Transition[T]
	return zero, false
}

// HasCycle detects if there are any cycles in the state machine.
func (sm *StateMachine[T]) HasCycle() bool {
	states := sm.GetAllStates()
	visited := make(map[T]bool)
	recStack := make(map[T]bool)

	for _, state := range states {
		if !visited[state] {
			if sm.hasCycleDFS(state, visited, recStack) {
				return true
			}
		}
	}

	return false
}

// hasCycleDFS performs depth-first search to detect cycles.
func (sm *StateMachine[T]) hasCycleDFS(state T, visited, recStack map[T]bool) bool {
	visited[state] = true
	recStack[state] = true

	// Get all states reachable from current state
	for _, transition := range sm.transitions {
		if transition.From == state {
			neighbor := transition.To
			if !visited[neighbor] {
				if sm.hasCycleDFS(neighbor, visited, recStack) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}
	}

	recStack[state] = false
	return false
}

// String returns a string representation of the state machine.
func (sm *StateMachine[T]) String() string {
	return fmt.Sprintf("StateMachine[current: %v, transitions: %d]",
		sm.currentState, len(sm.transitions))
}
