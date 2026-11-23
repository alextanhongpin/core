package welford_test

import (
	"fmt"

	"github.com/alextanhongpin/core/types/welford"
)

func ExampleWelfordAnomalyDetector() {
	d := welford.NewDetector(3) // flag values more than 3 standard deviations out
	for _, ms := range []float64{101, 99, 102, 98, 100, 250, 101} {
		if d.Update(ms) {
			fmt.Println("anomaly:", ms)
		}
	}
	fmt.Printf("n=%d mean=%.1f\n", d.Count(), d.Mean())
	// Output:
	// anomaly: 250
	// n=7 mean=121.6
}
