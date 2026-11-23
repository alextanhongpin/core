package dataloader_test

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/core/sync/dataloader"
)

func Example() {
	ctx := context.Background()
	fetch := func(ctx context.Context, ids []int) (map[int]string, error) {
		out := make(map[int]string, len(ids))
		for _, id := range ids {
			out[id] = fmt.Sprintf("user-%d", id)
		}
		return out, nil
	}
	dl, stop, err := dataloader.New(ctx, fetch, dataloader.Config{})
	if err != nil {
		panic(err)
	}
	defer stop()

	name, err := dl.LoadContext(ctx, 7)
	fmt.Println(name, err)
	// Output: user-7 <nil>
}
