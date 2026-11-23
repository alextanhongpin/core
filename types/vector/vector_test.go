package vector_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/alextanhongpin/core/types/vector"
)

func TestVector(t *testing.T) {
	a := vector.Vector{3, 4}
	if got := a.Magnitude(); got != 5 {
		t.Fatalf("magnitude = %v", got)
	}
	if got := a.Dot(vector.Vector{1, 2}); got != 11 {
		t.Fatalf("dot = %v", got)
	}
	tests := []struct {
		b    vector.Vector
		want float64
	}{
		{vector.Vector{6, 8}, 1},
		{vector.Vector{-3, -4}, -1},
		{vector.Vector{-4, 3}, 0},
		{vector.Vector{0, 0}, 0},
	}
	for _, tt := range tests {
		if got := a.CosineSimilarity(tt.b); math.Abs(got-tt.want) > 1e-12 {
			t.Errorf("cos(%v, %v) = %v, want %v", a, tt.b, got, tt.want)
		}
	}
}

func TestLengthMismatchPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("want panic")
		}
	}()
	vector.Vector{1}.CosineSimilarity(vector.Vector{1, 2})
}

func ExampleVector_CosineSimilarity() {
	a := vector.Vector{1, 0}
	b := vector.Vector{1, 1}
	fmt.Printf("%.4f\n", a.CosineSimilarity(b))
	// Output: 0.7071
}
