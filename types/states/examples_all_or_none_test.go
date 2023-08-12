package states_test

import (
	"fmt"

	"github.com/alextanhongpin/core/types/states"
)

type Refund struct {
	Status        string
	FailureReason string
}

// If status is "failed", a failure reason must be provided.
// Otherwise, it should be empty.
func (r *Refund) IsValid() bool {
	return (r.Status == "failed") == (r.FailureReason != "")
}

func ExampleAllOrNone() {
	var r Refund
	r.Status = "success"

	fmt.Println(states.AllOrNone(
		r.Status == "failed",  // status is failed
		r.FailureReason != "", // failure reason is not empty
	))
	fmt.Println(r.IsValid())

	r.Status = "failed"
	fmt.Println(states.AllOrNone(
		r.Status == "failed",
		r.FailureReason != "",
	))
	fmt.Println(r.IsValid())
	// Output:
	// true
	// true
	// false
	// false
}
