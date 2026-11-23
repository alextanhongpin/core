package reservoir_test

import (
	"fmt"

	"github.com/alextanhongpin/core/types/reservoir"
)

func ExampleReservoirSampling() {
	r := reservoir.NewReservoirSampling[int](3)
	for i := range 1000 {
		r.Add(i)
	}
	fmt.Println(len(r.Sample()), r.Count())
	// Output: 3 1000
}
