// Run with go run ./example/fixed > report.html, then open report.html.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/alextanhongpin/core/ab"
)

func main() {
	experiment, err := ab.NewFixedTest("checkout")
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 10000; i++ {
		user := fmt.Sprint(i)
		arm, err := experiment.Expose(user)
		if err != nil {
			log.Fatal(err)
		}
		// Synthetic outcomes for demonstration only; separate hash from assignment.
		threshold := uint64(10)
		if arm == "treatment" {
			threshold = 13
		}
		if ab.Hash("outcome:"+user, 100) < threshold {
			if err := experiment.Convert(user); err != nil {
				log.Fatal(err)
			}
		}
	}
	if err := experiment.Report().WriteHTML(os.Stdout); err != nil {
		log.Fatal(err)
	}
}
