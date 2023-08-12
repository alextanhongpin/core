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
	sm := states.NewState(Pending,
		states.NewTransition("success", Pending, Success),
		states.NewTransition("failed", Pending, Failed),
	)

	// Already pending.
	fmt.Println(sm.Transition(Pending))

	fmt.Println(sm.IsValidTransition(Pending, Success))
	fmt.Println(sm.IsValidTransition(Pending, Failed))
	fmt.Println(sm.IsValidTransition(Success, Failed))
	fmt.Println(sm.IsValidTransition(Failed, Success))

	fmt.Println(sm.Transition(Success))
	fmt.Println(sm.State())
	fmt.Println(sm.Transition(Failed))
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
