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

	fmt.Println(sm.Validate(Pending, Success))
	fmt.Println(sm.Validate(Pending, Failed))
	fmt.Println(sm.Validate(Success, Failed))
	fmt.Println(sm.Validate(Failed, Success))

	fmt.Println(sm.Transition(Success))
	fmt.Println(sm.State())
	fmt.Println(sm.Transition(Failed))
	// Output:
	// false
	// success true
	// failed true
	//  false
	//  false
	// true
	// success
	// false
}
