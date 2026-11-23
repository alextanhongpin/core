package promise_test

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/core/sync/promise"
)

func Example() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := promise.New(ctx, func(ctx context.Context) (int, error) { return 1, nil })
	b := promise.Resolve(ctx, 2)

	vals, err := promise.All(a, b)
	fmt.Println(vals, err)

	// Deferred promises must be settled or aborted.
	d := promise.Deferred[string](ctx)
	d.Resolve("done")
	fmt.Println(d.Await())
	// Output:
	// [1 2] <nil>
	// done <nil>
}
