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

type Sequential struct {
	steps []Step
}

func NewSequential(steps ...Step) *Sequential {
	return &Sequential{
		steps: steps,
	}
}

func (s *Sequential) Current() (string, bool) {
	var stopIdx int

	// Check which steps are completed.
	for i, step := range s.steps {
		if !step.cond() {
			stopIdx = i
			break
		}
	}

	// All other steps after must be not completed.
	for _, step := range s.steps[stopIdx:] {
		if step.cond() {
			return "", false
		}
	}

	return s.steps[stopIdx].name, stopIdx > -1
}

// XOR returns true if at least n conditional is true.
// Useful for checking polymorphic conditions.
func XOR(n int, ts ...bool) bool {
	var success int
	for _, t := range ts {
		if t {
			success++
		}
	}

	return success == n
}

func XORFunc(n int, hs ...Handler) bool {
	var success int
	for _, h := range hs {
		if h() {
			success++
		}
	}

	return success == n
}

// AllOrNone returns true if all condition is true, or all
// is false.
// Useful to check if a set of fields are all set or none
// are set.
func AllOrNone(ts ...bool) bool {
	var success int
	for _, t := range ts {
		if t {
			success++
		}
	}

	return success == 0 || success == len(ts)
}

func AllOrNoneFunc(hs ...Handler) bool {
	var success int
	for _, h := range hs {
		if h() {
			success++
		}
	}

	return success == 0 || success == len(hs)
}
