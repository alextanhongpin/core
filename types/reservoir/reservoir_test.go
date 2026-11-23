package reservoir_test

import (
	"slices"
	"testing"

	"github.com/alextanhongpin/core/types/reservoir"
)

func TestSampleShorterThanK(t *testing.T) {
	r := reservoir.NewReservoirSampling[int](5)
	if got := r.Sample(); len(got) != 0 {
		t.Fatalf("empty sample = %v", got)
	}
	r.Add(1)
	r.Add(2)
	if got := r.Sample(); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("sample = %v, want [1 2]", got)
	}
	if r.Count() != 2 {
		t.Fatalf("count = %d", r.Count())
	}
}

func TestSampleIsCopy(t *testing.T) {
	r := reservoir.NewReservoirSampling[int](1)
	r.Add(1)
	s := r.Sample()
	s[0] = 99
	if r.Sample()[0] != 1 {
		t.Fatal("Sample exposed internal storage")
	}
}

func TestUniform(t *testing.T) {
	const n, k, trials = 10, 3, 20000
	counts := make([]int, n)
	for range trials {
		r := reservoir.NewReservoirSampling[int](k)
		for i := range n {
			r.Add(i)
		}
		s := r.Sample()
		if len(s) != k {
			t.Fatalf("len = %d", len(s))
		}
		for _, v := range s {
			counts[v]++
		}
	}
	want := float64(trials) * k / n
	for i, c := range counts {
		if d := float64(c) - want; d > want*0.1 || d < -want*0.1 {
			t.Errorf("item %d picked %d times, want about %.0f", i, c, want)
		}
	}
}

func TestNonPositiveKPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("want panic")
		}
	}()
	reservoir.NewReservoirSampling[int](0)
}
