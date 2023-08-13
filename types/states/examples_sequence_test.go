package states_test

import (
	"fmt"

	"github.com/alextanhongpin/core/types/states"
)

type Shipment struct {
	FirstMile bool
	MidMile   bool
	LastMile  bool
}

func ExampleSequential() {
	var s Shipment

	seq := states.NewSequence(
		states.NewStepFunc("first_mile_completed", func() bool { return s.FirstMile }),
		states.NewStepFunc("mid_mile_completed", func() bool { return s.MidMile }),
		states.NewStepFunc("last_mile_completed", func() bool { return s.LastMile }),
	)

	fmt.Println(seq.None())
	fmt.Println(seq.Pending())
	fmt.Println(seq.Valid())

	s.FirstMile = true
	fmt.Println(seq.Pending())

	s.MidMile = true
	fmt.Println(seq.Pending())

	s.LastMile = true
	fmt.Println(seq.Pending())
	fmt.Println(seq.All())
	fmt.Println(seq.Valid())

	// When the sequence is invalid.
	s.FirstMile = false
	s.MidMile = true
	s.LastMile = false
	fmt.Println(seq.Pending())
	fmt.Println(seq.Valid())
	// Output:
	// true
	// first_mile_completed
	// true
	// mid_mile_completed
	// last_mile_completed
	//
	// true
	// true
	//
	// false
}
