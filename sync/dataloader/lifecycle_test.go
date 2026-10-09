package dataloader_test

import (
	"context"
	"runtime"
	"testing"
	"testing/synctest"
	"time"

	"github.com/alextanhongpin/core/sync/dataloader"
)

func TestConfigSnapshotAndDefaults(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := &dataloader.Config{BatchSize: 1}
		dl, stop := dataloader.New(context.Background(), func(_ context.Context, keys []int) (map[int]int, error) {
			return map[int]int{keys[0]: keys[0]}, nil
		}, cfg)
		defer stop()
		cfg.BatchSize = -1
		dl.Config.BatchSize = -1
		got, err := dl.Load(42)
		if err != nil || got != 42 {
			t.Fatalf("Load = %d, %v", got, err)
		}
	})
}

func TestInvalidConfigPanicsSynchronously(t *testing.T) {
	for _, cfg := range []dataloader.Config{
		{BatchSize: -1}, {BatchInterval: -time.Second}, {BufferSize: -1},
	} {
		t.Run("invalid", func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected constructor panic")
				}
			}()
			_, stop := dataloader.New(t.Context(), func(context.Context, []int) (map[int]int, error) { return nil, nil }, &cfg)
			stop()
		})
	}
}

func TestBatchRetainsPendingResults(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		release := make(chan struct{})
		dl, stop := dataloader.New(context.Background(), func(_ context.Context, keys []int) (map[int]int, error) {
			close(started)
			<-release
			runtime.GC()
			return map[int]int{keys[0]: 42}, nil
		}, &dataloader.Config{BatchSize: 1})
		done := make(chan struct{})
		go func() { defer close(done); _, _ = dl.Load(1) }()
		<-started
		stopped := make(chan struct{})
		go func() { stop(); close(stopped) }()
		<-done
		synctest.Wait()
		runtime.GC()
		close(release)
		<-stopped
	})
}
