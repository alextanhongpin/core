package states

type Transition[T comparable] struct {
	Name string
	From T
	To   T
}

func NewTransition[T comparable](name string, from, to T) Transition[T] {
	return Transition[T]{
		Name: name,
		From: from,
		To:   to,
	}
}

type State[T comparable] struct {
	state  T
	states []Transition[T]
}

func NewState[T comparable](initial T, states ...Transition[T]) *State[T] {
	return &State[T]{
		state:  initial,
		states: states,
	}
}

func (s *State[T]) State() T {
	return s.state
}

func (s *State[T]) Exec(name string) (ok bool, found bool) {
	for _, st := range s.states {
		if st.Name == name {
			return s.Transition(st.To), true
		}
	}

	return false, false
}

func (s *State[T]) TransitionFunc(to T, fn func() error) (bool, error) {
	if !s.CanTransition(to) {
		return false, nil
	}

	if err := fn(); err != nil {
		return false, err
	}

	s.state = to
	return true, nil
}

func (s *State[T]) Transition(to T) bool {
	if s.CanTransition(to) {
		s.state = to
		return true
	}

	return false
}

func (s *State[T]) CanTransition(to T) bool {
	if s.state == to {
		return false
	}

	return s.IsValidTransition(s.state, to)
}

func (s *State[T]) IsValidTransition(from, to T) bool {
	for _, s := range s.states {
		if s.From == from && s.To == to {
			return true
		}
	}

	return false
}
