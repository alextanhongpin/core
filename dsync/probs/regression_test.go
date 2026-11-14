package probs_test

import (
	"testing"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/dbtx/testing/redistest"
)

func TestHyperLogLogVariadicValues(t *testing.T) {
	hll := probs.MustNewHyperLogLog(redistest.Client(t))
	if _, err := hll.Add(ctx, t.Name(), "a", "b", "c"); err != nil {
		t.Fatal(err)
	}
	if n, err := hll.Count(ctx, t.Name()); err != nil || n != 3 {
		t.Fatalf("%d %v", n, err)
	}
}

// A missing source used to create an empty destination as a side effect.
func TestMissingSketchSourceReturnsWithoutRecursiveRetry(t *testing.T) {
	client := redistest.Client(t)
	cms := probs.MustNewCountMinSketch(client)
	if _, err := cms.Merge(ctx, t.Name(), "missing-source"); err == nil {
		t.Fatal("missing source accepted")
	}
	if n := client.Exists(ctx, t.Name()).Val(); n != 0 {
		t.Fatal("destination created although the merge failed")
	}
	if _, err := cms.MergeWithWeight(ctx, t.Name(), map[string]int64{"missing-source": 1}); err == nil {
		t.Fatal("missing source accepted")
	}
	if n := client.Exists(ctx, t.Name()).Val(); n != 0 {
		t.Fatal("destination created although the merge failed")
	}
}

// Merging into a missing destination used to create it with the default
// dimensions, so merging sketches made with InitByDim failed with
// "width/depth is not equal" and left a mismatched destination behind.
func TestCMSMergeIntoMissingDestUsesSourceDimensions(t *testing.T) {
	cms := probs.MustNewCountMinSketch(redistest.Client(t))
	src1, src2 := t.Name()+":1", t.Name()+":2"
	for _, k := range []string{src1, src2} {
		if _, _, err := cms.InitByDim(ctx, k, 100, 3); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := cms.IncrBy(ctx, src1, map[string]int64{"a": 2}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := cms.IncrBy(ctx, src2, map[string]int64{"a": 3}); err != nil {
		t.Fatal(err)
	}

	dst := t.Name() + ":dst"
	if _, err := cms.Merge(ctx, dst, src1, src2); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if got, err := cms.Query(ctx, dst, "a"); err != nil || got[0] != 5 {
		t.Fatalf("Query = %v, %v; want [5]", got, err)
	}

	weighted := t.Name() + ":weighted"
	if _, err := cms.MergeWithWeight(ctx, weighted, map[string]int64{src1: 1, src2: 2}); err != nil {
		t.Fatalf("MergeWithWeight: %v", err)
	}
	if got, err := cms.Query(ctx, weighted, "a"); err != nil || got[0] != 8 {
		t.Fatalf("Query = %v, %v; want [8]", got, err)
	}
}

func TestTopKSingleSlot(t *testing.T) {
	top := probs.MustNewTopK(redistest.Client(t))
	if _, err := top.Create(ctx, t.Name(), 1); err != nil {
		t.Fatal(err)
	}
	if _, err := top.Create(ctx, t.Name()+"-invalid", 0); err == nil {
		t.Fatal("invalid k accepted")
	}
}
