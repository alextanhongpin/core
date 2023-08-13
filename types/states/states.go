package states

type Handler func() bool

type Step struct {
	name string
	cond Handler
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

func (s *Sequence) None() bool {
	return s.success() == 0
}

func (s *Sequence) All() bool {
	return s.success() == len(s.steps)
}

func (s *Sequence) Pending() string {
	if !s.Valid() {
		return ""
	}

	for _, step := range s.steps {
		if !step.cond() {
			return step.name
		}
	}

	return ""
}

func (s *Sequence) Valid() bool {
	for i, step := range s.steps {
		if step.cond() {
			continue
		}

		for _, step := range s.steps[i+1:] {
			if step.cond() {
				return false
			}
		}
	}

	return true
}

func (s *Sequence) success() int {
	var count int
	for _, step := range s.steps {
		if step.cond() {
			count++
		}
	}

	return count
}
