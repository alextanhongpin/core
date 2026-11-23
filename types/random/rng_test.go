package random_test

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/types/random"
)

func TestBetweenWideRange(t *testing.T) {
	rng := random.New().WithSeed(1)
	sawNonMin := false
	for range 100 {
		if v := rng.IntBetween(math.MinInt, math.MaxInt); v != math.MinInt {
			sawNonMin = true
		}
		d := rng.DurationBetween(time.Duration(math.MinInt64), time.Duration(math.MaxInt64))
		if d == time.Duration(math.MaxInt64) {
			t.Fatal("max must be exclusive")
		}
	}
	if !sawNonMin {
		t.Fatal("IntBetween over the full int range always returned min")
	}
}

func TestBetweenBounds(t *testing.T) {
	rng := random.New().WithSeed(1)
	for range 1000 {
		if v := rng.IntBetween(-3, 3); v < -3 || v >= 3 {
			t.Fatalf("IntBetween out of range: %d", v)
		}
	}
	if got := rng.IntBetween(5, 1); got != 5 {
		t.Fatalf("max <= min should return min, got %d", got)
	}
}

func TestStringMultiByteCharset(t *testing.T) {
	s := random.New().WithSeed(1).String(10, "日本")
	n := 0
	for _, r := range s {
		if r != '日' && r != '本' {
			t.Fatalf("unexpected rune %q in %q", r, s)
		}
		n++
	}
	if n != 10 {
		t.Fatalf("want 10 runes, got %d", n)
	}
}

func TestSampleLargerThanInput(t *testing.T) {
	if got := random.Sample([]int{1, 2, 3}, 10); len(got) != 3 {
		t.Fatalf("want 3 items, got %v", got)
	}
}

// Run with -race: package-level functions share one RNG.
func TestConcurrentUse(t *testing.T) {
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 100 {
				random.IntBetween(0, 10)
				random.AlphaNumeric(8)
				random.Shuffle([]int{1, 2, 3})
			}
		})
	}
	wg.Wait()
}
