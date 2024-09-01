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
		fmt.Println("has next?", step.Name(), ok)
		fmt.Printf("step %d: status=%s valid=%t\n", i+1, seq.Status(), seq.Valid())
		fmt.Println()

		status[i%len(status)] = true
	}

	status[1] = false
	step, ok := seq.Next()
	fmt.Println("has next?", step.Name(), ok)
	fmt.Printf("invalid: status=%s valid=%t\n", seq.Status(), seq.Valid())
	fmt.Println()

	// Output:
	// has next? first_mile_completed true
	// step 1: status=idle valid=true
	//
	// has next? mid_mile_completed true
	// step 2: status=pending valid=true
	//
	// has next? last_mile_completed true
	// step 3: status=pending valid=true
	//
	// has next?  false
	// step 4: status=success valid=true
	//
	// has next?  false
	// invalid: status=unknown valid=false
}
