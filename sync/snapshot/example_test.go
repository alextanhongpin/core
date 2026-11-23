package snapshot_test

import (
	"fmt"
	"log"

	"github.com/alextanhongpin/core/sync/snapshot"
)

func Example() {
	s, stop, err := snapshot.New(snapshot.Config{
		Policies: []snapshot.Policy{{Changes: 10}},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stop()

	notifications := s.Chan() // subscribe before recording changes
	s.Add(10)
	p := <-notifications
	fmt.Println("persist after", p.Changes, "changes")
	// Output: persist after 10 changes
}
