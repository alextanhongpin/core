package dataloader_test

import (
	"context"
	"github.com/alextanhongpin/core/sync/dataloader"
	"testing"
	"time"
)

func TestNewRejectsInvalidInputsBeforeStarting(t *testing.T) {
	fn := func(context.Context, []int) (map[int]int, error) { return nil, nil }
	for _, cfg := range []dataloader.Config{{BatchSize: -1}, {BatchInterval: -time.Second}, {BufferSize: -1}} {
		dl, stop, err := dataloader.New(t.Context(), fn, cfg)
		if err == nil || dl != nil || stop != nil {
			t.Fatalf("accepted invalid configuration: %+v", cfg)
		}
	}
	if dl, stop, err := dataloader.New[int, int](t.Context(), nil, dataloader.Config{}); err == nil || dl != nil || stop != nil {
		t.Fatal("nil batch function accepted")
	}
	dl, stop, err := dataloader.New(t.Context(), fn, dataloader.Config{})
	if err != nil || dl == nil {
		t.Fatal(err)
	}
	stop()
}
