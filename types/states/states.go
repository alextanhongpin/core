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

func (s *Sequence) NotStarted() bool {
	return s.Valid() && s.next() == 0
}

func (s *Sequence) Done() bool {
	return s.Valid() && s.next() == len(s.steps)
}

func (s *Sequence) Pending() bool {
	return s.Valid() && !s.NotStarted() && !s.Done()
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
