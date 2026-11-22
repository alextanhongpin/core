package broadcast_test

import (
	"fmt"

	"github.com/alextanhongpin/core/sync/broadcast"
)

func Example() {
	b, stop := broadcast.New[int]()
	defer stop()

	ch, unsubscribe := b.Subscribe()
	defer unsubscribe()

	go b.Send(42)
	fmt.Println(<-ch)
	// Output: 42
}
