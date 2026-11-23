package states

import (
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrUnknownTransition is returned when no transition with the given
	// name starts at the current state.
	ErrUnknownTransition = errors.New("states: unknown transition")
	// ErrInvalidTransition is returned when no declared transition leads
	// from the current state to the target state.
	ErrInvalidTransition = errors.New("states: invalid transition")
)

// Transition is a named, directed edge between two states.
type Transition[T comparable] struct {
	Name string
	From T
	To   T
}

// NewTransition returns a transition named name from one state to another.
func NewTransition[T comparable](name string, from, to T) Transition[T] {
	return Transition[T]{Name: name, From: from, To: to}
}

// String returns "name: from -> to".
func (t Transition[T]) String() string {
	return fmt.Sprintf("%s: %v -> %v", t.Name, t.From, t.To)
}

// Listener is called after the state machine changes state.
type Listener[T comparable] func(t Transition[T])

// StateMachine holds a current state and moves between states only along
// declared transitions. The same name may be used for transitions from
// several states (for example "cancel"). A self-transition is allowed only
// when it is declared.
//
// A StateMachine is safe for concurrent use. Listeners run after the state
// changes, outside the internal lock, so they may call the machine's methods;
// listeners for concurrent transitions may run concurrently.
type StateMachine[T comparable] struct {
	mu          sync.Mutex
	state       T
	transitions []Transition[T]
	listeners   []Listener[T]
}

// NewStateMachine returns a machine in the initial state with the given
// transitions.
func NewStateMachine[T comparable](initial T, transitions ...Transition[T]) *StateMachine[T] {
	return &StateMachine[T]{
		state:       initial,
		transitions: append([]Transition[T](nil), transitions...),
	}
}

// State returns the current state.
func (sm *StateMachine[T]) State() T {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state
}

// AddTransition declares more transitions.
func (sm *StateMachine[T]) AddTransition(ts ...Transition[T]) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.transitions = append(sm.transitions, ts...)
}

// AddListener registers fn to be called after every state change.
func (sm *StateMachine[T]) AddListener(fn Listener[T]) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.listeners = append(sm.listeners, fn)
}

// Execute fires the transition with the given name from the current state.
// It returns an error wrapping ErrUnknownTransition if there is none.
func (sm *StateMachine[T]) Execute(name string) error {
	return sm.ExecuteFunc(name, nil)
}

// ExecuteFunc is like Execute, but first runs fn and only changes state if fn
// returns nil. fn runs without the lock held; if another goroutine changes
// the state while fn runs, ExecuteFunc returns an error wrapping
// ErrUnknownTransition (unless the transition is valid from the new state)
// and the state is left as the other goroutine set it.
func (sm *StateMachine[T]) ExecuteFunc(name string, fn func() error) error {
	sm.mu.Lock()
	_, ok := sm.byName(name)
	from := sm.state
	sm.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w %q from %v", ErrUnknownTransition, name, from)
	}
	if fn != nil {
		if err := fn(); err != nil {
			return err
		}
	}

	sm.mu.Lock()
	t, ok := sm.byName(name)
	if !ok {
		from := sm.state
		sm.mu.Unlock()
		return fmt.Errorf("%w %q from %v", ErrUnknownTransition, name, from)
	}
	return sm.apply(t) // unlocks
}

// TransitionTo moves to target along the first declared transition from the
// current state to target. It returns an error wrapping ErrInvalidTransition
// if there is none.
func (sm *StateMachine[T]) TransitionTo(target T) error {
	sm.mu.Lock()
	t, ok := sm.to(target)
	if !ok {
		from := sm.state
		sm.mu.Unlock()
		return fmt.Errorf("%w from %v to %v", ErrInvalidTransition, from, target)
	}
	return sm.apply(t) // unlocks
}

// apply sets the state to t.To, releases the lock, then calls listeners.
func (sm *StateMachine[T]) apply(t Transition[T]) error {
	sm.state = t.To
	listeners := sm.listeners
	sm.mu.Unlock()
	for _, fn := range listeners {
		fn(t)
	}
	return nil
}

// CanTransitionTo reports whether a declared transition leads from the
// current state to target.
func (sm *StateMachine[T]) CanTransitionTo(target T) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	_, ok := sm.to(target)
	return ok
}

// Available returns the transitions that start at the current state, in
// declaration order.
func (sm *StateMachine[T]) Available() []Transition[T] {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	var out []Transition[T]
	for _, t := range sm.transitions {
		if t.From == sm.state {
			out = append(out, t)
		}
	}
	return out
}

// States returns every state that appears in a transition, plus the current
// state, in order of first appearance (current state first).
func (sm *StateMachine[T]) States() []T {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	seen := map[T]bool{sm.state: true}
	out := []T{sm.state}
	for _, t := range sm.transitions {
		for _, s := range [2]T{t.From, t.To} {
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out
}

// HasCycle reports whether any state can be reached again from itself,
// including through a declared self-transition.
func (sm *StateMachine[T]) HasCycle() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	next := make(map[T][]T)
	for _, t := range sm.transitions {
		next[t.From] = append(next[t.From], t.To)
	}
	const (
		unvisited = iota
		active
		done
	)
	mark := make(map[T]int)
	var visit func(T) bool
	visit = func(s T) bool {
		mark[s] = active
		for _, n := range next[s] {
			switch mark[n] {
			case active:
				return true
			case unvisited:
				if visit(n) {
					return true
				}
			}
		}
		mark[s] = done
		return false
	}
	for _, t := range sm.transitions {
		if mark[t.From] == unvisited && visit(t.From) {
			return true
		}
	}
	return false
}

func (sm *StateMachine[T]) byName(name string) (Transition[T], bool) {
	for _, t := range sm.transitions {
		if t.Name == name && t.From == sm.state {
			return t, true
		}
	}
	return Transition[T]{}, false
}

func (sm *StateMachine[T]) to(target T) (Transition[T], bool) {
	for _, t := range sm.transitions {
		if t.From == sm.state && t.To == target {
			return t, true
		}
	}
	return Transition[T]{}, false
}
