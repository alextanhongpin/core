package states_test

import (
	"fmt"

	"github.com/alextanhongpin/core/types/states"
)

type Status string

var (
	Pending Status = "pending"
	Success Status = "success"
	Failed  Status = "failed"
)

func ExampleTransition() {
	sm := states.NewStateMachine(Pending,
		states.NewTransition("resolve", Pending, Success),
		states.NewTransition("reject", Pending, Failed),
	)

	// Already pending.
	fmt.Println(sm.TransitionTo(Pending, "stay"))

	fmt.Println(sm.IsValidTransition(Pending, Success))
	fmt.Println(sm.IsValidTransition(Pending, Failed))
	fmt.Println(sm.IsValidTransition(Success, Failed))
	fmt.Println(sm.IsValidTransition(Failed, Success))

	fmt.Println(sm.TransitionTo(Success, "resolve"))
	fmt.Println(sm.State())
	fmt.Println(sm.TransitionTo(Failed, "reject"))
	// Output:
	// false
	// true
	// true
	// false
	// false
	// true
	// success
	// false
}
