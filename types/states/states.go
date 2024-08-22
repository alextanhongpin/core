// package states ensure all steps are completed before proceeding to the next
// step.
package states

type Handler func() bool

type Step struct {
	name string
	cond Handler
}

func (s Step) Name() string {
	return s.name
}

func NewStep(name string, bs bool) Step {
	return Step{
		name: name,
		cond: func() bool {
			return bs
		},
	}
}

func NewStepFunc(name string, cond Handler) Step {
	return Step{
		name: name,
		cond: cond,
	}
}

type Sequence struct {
	steps []Step
}

func NewSequence(steps ...Step) *Sequence {
	return &Sequence{
		steps: steps,
	}
}

func (s *Sequence) Status() Status {
	if !s.Valid() {
		return -1
	}

	switch s.next() {
	case 0:
		return NotStarted
	case len(s.steps):
		return Success
	default:
		return Pending
	}
}

func (s *Sequence) Next() (Step, bool) {
	if s.Valid() {
		if s.next() == len(s.steps) {
			return Step{}, false
		}

		return s.steps[s.next()], true
	}

	return Step{}, false
}

func (s *Sequence) Valid() bool {
	p := s.progress()
	i := s.next()
	// We can calculate the cumulative by taking the next bit - 1
	return p == (1<<i)-1
}

// next returns the next pending step.
func (s *Sequence) next() int {
	for i, step := range s.steps {
		if !step.cond() {
			return i
		}
	}

	return len(s.steps)
}

// progress checks the current progress of the sequence.
func (s *Sequence) progress() int {
	var progress int
	for i, step := range s.steps {
		if step.cond() {
			progress |= 1 << i
		}
	}

	return progress
}
