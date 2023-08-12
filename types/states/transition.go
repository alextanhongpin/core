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

func (s *State[T]) Transition(to T) bool {
	if s.state == to {
		return false
	}

	_, ok := s.Validate(s.state, to)
	if ok {
		s.state = to
		return true
	}

	return false
}

func (s *State[T]) Validate(from, to T) (string, bool) {
	for _, s := range s.states {
		if s.From == from && s.To == to {
			return s.Name, true
		}
	}

	return "", false
}
