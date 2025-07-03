package states_test

import (
	"fmt"

	"github.com/alextanhongpin/core/types/states"
)

func ExampleXOR() {
	type Party struct {
		Person       bool
		Organization bool
	}

	var p Party
	p.Person = true

	fmt.Println(states.XOR(true, p.Person, p.Organization))
	// Output:
	// true
}
