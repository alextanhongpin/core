package states_test

import (
	"fmt"

	"github.com/alextanhongpin/core/types/states"
)

func ExampleSequential() {
	status := []bool{false, false, false}

	seq := states.NewSequence(
		states.NewStepFunc("first_mile_completed", func() bool { return status[0] }),
		states.NewStepFunc("mid_mile_completed", func() bool { return status[1] }),
		states.NewStepFunc("last_mile_completed", func() bool { return status[2] }),
	)

	for i := range 4 {
		step, ok := seq.Next()
		fmt.Println(step.Name(), ok)
		fmt.Printf("step %d: not_started=%t, pending=%t, done=%t valid=%t\n", i+1, seq.NotStarted(), seq.Pending(), seq.Done(), seq.Valid())
		fmt.Println()

		status[i%len(status)] = true
	}

	status[1] = false
	step, ok := seq.Next()
	fmt.Println(step.Name(), ok)
	fmt.Printf("invalid: not_started=%t, pending=%t, done=%t valid=%t\n", seq.NotStarted(), seq.Pending(), seq.Done(), seq.Valid())
	fmt.Println()

	// Output:
	// first_mile_completed true
	// step 1: not_started=true, pending=false, done=false valid=true
	//
	// mid_mile_completed true
	// step 2: not_started=false, pending=true, done=false valid=true
	//
	// last_mile_completed true
	// step 3: not_started=false, pending=true, done=false valid=true
	//
	//  false
	// step 4: not_started=false, pending=false, done=true valid=true
	//
	//  false
	// invalid: not_started=false, pending=false, done=false valid=false
}
