package rate

import (
	"math"
	"testing"
	"time"
)

func TestClockRegressionDoesNotDoubleDecay(t *testing.T) {
	r := Per(time.Second)
	now := time.Unix(100, 0)
	r.Now = func() time.Time { return now }
	r.Add(10)
	now = now.Add(-time.Second)
	if got := r.Count(); got != 10 {
		t.Fatal(got)
	}
	now = now.Add(time.Second)
	if got := r.Count(); got != 10 {
		t.Fatalf("returning to previous time decayed count: %v", got)
	}
}
func TestLimiterRejectsNonFiniteLimits(t *testing.T) {
	for _, limit := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		t.Run("nonfinite", func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			NewLimiter(limit)
		})
	}
}
