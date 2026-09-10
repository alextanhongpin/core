package dataloader_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
)

func BenchmarkDataLoader(b *testing.B) {
	ctx := b.Context()
	n := 1000
	arr := make([]string, n)
	for i := range 1000 {
		arr[i] = fmt.Sprint(rand.N(10_000))
	}

	for b.Loop() {
		dl, stop := newDataloader(ctx, func(ctx context.Context, keys []string) (map[string]int, error) {
			return newBatchFn(ctx, nil)
		})
		defer stop()

		_, _ = dl.LoadMany(arr...)
	}
}
