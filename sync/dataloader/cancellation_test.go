package dataloader_test

import (
	"context"
	"errors"
	"github.com/alextanhongpin/core/sync/dataloader"
	"testing"
)

func TestCallerCancellationPreservesSharedLoad(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	dl, stop := dataloader.New(context.Background(), func(ctx context.Context, keys []int) (map[int]int, error) {
		close(started)
		select {
		case <-release:
			return map[int]int{1: 42}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}, &dataloader.Config{BatchSize: 1})
	defer stop()
	ctx, cancel := context.WithCancel(context.Background())
	canceled := make(chan error, 1)
	fn := dataloader.Func(func(context.Context, int) (int, error) { t.Error("canceled transform invoked"); return 0, nil }, dl)
	go func() { _, err := fn(ctx, 1); canceled <- err }()
	<-started
	cancel()
	if err := <-canceled; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	close(release)
	if got, err := dl.LoadContext(t.Context(), 1); err != nil || got != 42 {
		t.Fatalf("shared load: %d %v", got, err)
	}
}
func TestCanceledCallerDoesNotStartLoad(t *testing.T) {
	dl, stop := dataloader.New(context.Background(), func(context.Context, []int) (map[int]int, error) { t.Error("canceled load invoked"); return nil, nil }, nil)
	defer stop()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := dl.LoadContext(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
