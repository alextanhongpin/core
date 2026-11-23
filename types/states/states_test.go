package states_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/alextanhongpin/core/types/states"
	"github.com/go-openapi/testify/assert"
)

func TestCounting(t *testing.T) {
	is := assert.New(t)

	is.True(states.ExactlyOne(true, false))
	is.False(states.ExactlyOne(true, true, true), "not a parity check")
	is.False(states.ExactlyOne())

	is.True(states.ExactlyN(2, true, false, true))

	// Regression: AtLeastN(0) used to be false with no conditions.
	is.True(states.AtLeastN(0))
	is.True(states.AtLeastN(2, true, false, true))
	is.False(states.AtLeastN(3, true, false, true))

	// Regression: AtMostN(-1) used to be true with no conditions.
	is.False(states.AtMostN(-1))
	is.True(states.AtMostN(1, true, false))
	is.False(states.AtMostN(1, true, true))

	is.True(states.AllOrNone())
	is.True(states.AllOrNone(false, false))
	is.True(states.AllOrNone(true, true))
	is.False(states.AllOrNone(true, false))

	is.False(states.Majority())
	is.False(states.Majority(true, false), "tie")
	is.True(states.Majority(true, true, false))
}

func TestSequence(t *testing.T) {
	is := assert.New(t)
	seq := func(done ...bool) *states.Sequence {
		steps := make([]states.Step, len(done))
		for i, d := range done {
			steps[i] = states.NewStep(fmt.Sprint(i), d)
		}
		return states.NewSequence(steps...)
	}

	is.Equal(states.Success, seq().Status())
	is.Equal(states.Idle, seq(false, false).Status())
	is.Equal(states.Pending, seq(true, false).Status())
	is.Equal(states.Success, seq(true, true).Status())
	is.Equal(states.Failed, seq(false, true).Status())
	is.False(seq(false, true).IsValid())

	next, ok := seq(true, false, false).Next()
	is.True(ok)
	is.Equal("1", next.Name())
	_, ok = seq(true, true).Next()
	is.False(ok)
	_, ok = seq(false, true).Next()
	is.False(ok)

	is.Equal(1, seq(false, true).CompletedSteps())
	is.Equal("pending [x] 0 [ ] 1", seq(true, false).String())
}

// Regression: Progress used a bitmask, so sequences with 64 or more steps
// reported the wrong status.
func TestSequenceManySteps(t *testing.T) {
	steps := make([]states.Step, 100)
	for i := range steps {
		steps[i] = states.NewStep(fmt.Sprint(i), true)
	}
	assert.Equal(t, states.Success, states.NewSequence(steps...).Status())
}

// Regression: conditions were evaluated several times per call, so a
// changing condition could produce an inconsistent result.
func TestSequenceEvaluatesOnce(t *testing.T) {
	calls := 0
	seq := states.NewSequence(states.NewStepFunc("a", func() bool {
		calls++
		return true
	}))
	seq.Status()
	assert.Equal(t, 1, calls)
}

type order string

const (
	pending   order = "pending"
	paid      order = "paid"
	shipped   order = "shipped"
	cancelled order = "cancelled"
)

func newOrder() *states.StateMachine[order] {
	return states.NewStateMachine(pending,
		states.NewTransition("pay", pending, paid),
		states.NewTransition("ship", paid, shipped),
		states.NewTransition("cancel", pending, cancelled),
		states.NewTransition("cancel", paid, cancelled),
	)
}

func TestStateMachine(t *testing.T) {
	is := assert.New(t)
	sm := newOrder()

	is.ErrorIs(sm.Execute("ship"), states.ErrUnknownTransition)
	is.NoError(sm.Execute("pay"))
	is.Equal(paid, sm.State())

	// Regression: Execute used the first transition with the name, so
	// "cancel" from paid failed because the first one starts at pending.
	is.NoError(sm.Execute("cancel"))
	is.Equal(cancelled, sm.State())

	is.ErrorIs(sm.TransitionTo(paid), states.ErrInvalidTransition)
	is.Equal([]order{cancelled, pending, paid, shipped}, sm.States())
	is.False(sm.HasCycle())
}

func TestStateMachineSelfTransition(t *testing.T) {
	is := assert.New(t)
	sm := states.NewStateMachine("a", states.NewTransition("retry", "a", "a"))
	// Regression: declared self-transitions were rejected.
	is.True(sm.CanTransitionTo("a"))
	is.NoError(sm.Execute("retry"))
	is.True(sm.HasCycle())
	is.False(states.NewStateMachine("a").CanTransitionTo("a"))
}

func TestStateMachineExecuteFunc(t *testing.T) {
	is := assert.New(t)
	sm := newOrder()
	boom := errors.New("boom")
	is.ErrorIs(sm.ExecuteFunc("pay", func() error { return boom }), boom)
	is.Equal(pending, sm.State())

	ran := false
	is.ErrorIs(sm.ExecuteFunc("ship", func() error { ran = true; return nil }), states.ErrUnknownTransition)
	is.False(ran, "fn is not run for an unknown transition")

	is.NoError(sm.ExecuteFunc("pay", func() error { return nil }))
	is.Equal(paid, sm.State())
}

// Regression: listeners ran while state was being mutated without a lock;
// a listener calling State must not deadlock.
func TestStateMachineListener(t *testing.T) {
	is := assert.New(t)
	sm := newOrder()
	var got []string
	sm.AddListener(func(tr states.Transition[order]) {
		got = append(got, fmt.Sprintf("%s now %s", tr, sm.State()))
	})
	is.NoError(sm.Execute("pay"))
	is.Equal([]string{"pay: pending -> paid now paid"}, got)
}

func TestStateMachineConcurrent(t *testing.T) {
	sm := states.NewStateMachine(0,
		states.NewTransition("flip", 0, 1),
		states.NewTransition("flip", 1, 0),
	)
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			_ = sm.Execute("flip")
			_ = sm.State()
			_ = sm.Available()
		})
	}
	wg.Wait()
}

func ExampleExactlyOne() {
	var person, organization = true, false
	fmt.Println(states.ExactlyOne(person, organization))
	// Output: true
}

func ExampleSequence() {
	seq := states.NewSequence(
		states.NewStep("verify email", true),
		states.NewStep("complete profile", false),
		states.NewStep("add payment", false),
	)
	next, _ := seq.Next()
	fmt.Println(seq.Status(), next.Name())
	// Output: pending complete profile
}

func ExampleStateMachine() {
	sm := newOrder()
	sm.AddListener(func(t states.Transition[order]) { fmt.Println(t) })
	_ = sm.Execute("pay")
	err := sm.Execute("pay")
	fmt.Println(errors.Is(err, states.ErrUnknownTransition))
	// Output:
	// pay: pending -> paid
	// true
}
