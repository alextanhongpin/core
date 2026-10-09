package probs_test

import (
	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"testing"
)

func TestHyperLogLogVariadicValues(t *testing.T) {
	hll := probs.NewHyperLogLog(redistest.Client(t))
	if _, err := hll.Add(ctx, t.Name(), "a", "b", "c"); err != nil {
		t.Fatal(err)
	}
	if n, err := hll.Count(ctx, t.Name()); err != nil || n != 3 {
		t.Fatalf("%d %v", n, err)
	}
}
func TestMissingSketchSourceReturnsWithoutRecursiveRetry(t *testing.T) {
	cms := probs.NewCountMinSketch(redistest.Client(t))
	if _, err := cms.Merge(ctx, t.Name(), "missing-source"); err == nil {
		t.Fatal("missing source accepted")
	}
}
func TestTopKSingleSlot(t *testing.T) {
	top := probs.NewTopK(redistest.Client(t))
	if _, err := top.Create(ctx, t.Name(), 1); err != nil {
		t.Fatal(err)
	}
	if _, err := top.Create(ctx, t.Name()+"-invalid", 0); err == nil {
		t.Fatal("invalid k accepted")
	}
}
