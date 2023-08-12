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

	seq := states.NewSequential(
		states.NewStepFunc("has_first_mile", func() bool { return s.FirstMile }),
		states.NewStepFunc("has_mid_mile", func() bool { return s.MidMile }),
		states.NewStepFunc("has_last_mile", func() bool { return s.LastMile }),
	)

	fmt.Println(seq.Current())

	s.FirstMile = true
	fmt.Println(seq.Current())

	s.MidMile = true
	fmt.Println(seq.Current())

	s.LastMile = true
	fmt.Println(seq.Current())

	// When the sequence is invalid.
	s.FirstMile = false
	s.MidMile = true
	s.LastMile = false
	fmt.Println(seq.Current())
	// Output:
	// has_first_mile true
	// has_mid_mile true
	// has_last_mile true
	//  false
	//  false
}
