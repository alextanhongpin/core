package backoff_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/backoff"
)

func TestConstant(t *testing.T) {
	for _, period := range []time.Duration{-1, 0, time.Second} {
		for _, index := range []int{-1, 0, 1, 100} {
			want := max(period, 0)
			if got := backoff.NewConstant(period).At(index); got != want {
				t.Fatalf("period=%v index=%d: got %v want %v", period, index, got, want)
			}
		}
	}
}

func TestLinear(t *testing.T) {
	for _, tc := range []struct {
		period time.Duration
		index  int
		want   time.Duration
	}{
		{-1, 1, 0}, {0, 1, 0}, {time.Second, -1, 0}, {time.Second, 0, 0},
		{time.Second, 1, time.Second}, {time.Second, 3, 3 * time.Second},
		{time.Duration(math.MaxInt64), 1, time.Duration(math.MaxInt64)},
		{time.Duration(math.MaxInt64), 2, time.Duration(math.MaxInt64)},
	} {
		if got := backoff.NewLinear(tc.period).At(tc.index); got != tc.want {
			t.Fatalf("%+v: got %v", tc, got)
		}
	}
}

func TestExponential(t *testing.T) {
	for _, tc := range []struct {
		base, cap time.Duration
		index     int
		bound     time.Duration
	}{
		{-1, time.Second, 1, 0}, {0, time.Second, 1, 0}, {time.Second, -1, 1, 0}, {time.Second, 0, 1, 0},
		{time.Second, 3 * time.Second, -1, time.Second}, {time.Second, 3 * time.Second, 0, time.Second},
		{time.Second, 3 * time.Second, 1, 2 * time.Second}, {time.Second, 3 * time.Second, 2, 3 * time.Second},
		{time.Second, 3 * time.Second, 62, 3 * time.Second}, {time.Second, 3 * time.Second, 100, 3 * time.Second},
		{time.Duration(math.MaxInt64), time.Minute, 1, time.Minute}, {1, 1, 1, 1},
	} {
		for range 100 {
			got := backoff.NewExponential(tc.base, tc.cap).At(tc.index)
			if tc.bound == 0 {
				if got != 0 {
					t.Fatalf("%+v: got %v", tc, got)
				}
				continue
			}
			if got < 0 || got >= tc.bound {
				t.Fatalf("%+v: got %v", tc, got)
			}
		}
	}
}

func ExampleLinear_At() {
	policy := backoff.NewLinear(100 * time.Millisecond)
	for index := 1; index <= 3; index++ {
		fmt.Println(policy.At(index))
	}
	// Output:
	// 100ms
	// 200ms
	// 300ms
}
