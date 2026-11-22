package pipeline_test

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/core/sync/pipeline"
)

func Example() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	in := pipeline.SourceSlice(ctx, []int{1, 2, 3})
	doubled := pipeline.PipeCtx(ctx, in, func(v int) (int, bool) { return v * 2, true })
	fmt.Println(pipeline.Collect(doubled))
	// Output: [2 4 6]
}
