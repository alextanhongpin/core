package rate_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
)

func TestRate(t *testing.T) {
	now := time.Now()
	period := 5 * time.Second

	r := rate.NewRate(period)
	f := func(name string, d time.Duration, exp int64) {
		t.Run(name, func(t *testing.T) {
			r.Now = func() time.Time { return now.Add(d) }
			got := r.Inc(1)
			if exp != got {
				t.Fatalf("expected %d, got %d", exp, got)
			}
		})
	}

	f("first", 0, 1)
	f("second", 1*time.Second, 2)
	f("third", 2*time.Second, 3)
	f("fourth", 3*time.Second, 2)
	f("fifth", 4*time.Second, 2)
	f("reset", 5*time.Second, 2)
}

func TestResetRate(t *testing.T) {
	perSecond := rate.NewRate(time.Second)
	perSecond.Inc(1)
	perSecond.Reset()
	got := perSecond.Inc(0)
	exp := int64(0)
	if exp != got {
		t.Fatalf("expected %d, got %d,", exp, got)
	}
}

func TestErrors(t *testing.T) {
	now := time.Now()
	period := 5 * time.Second

	r := rate.NewErrors(period)
	f := func(name string, d time.Duration, n int64, exp float64) {
		t.Run(name, func(t *testing.T) {
			r.SetNow(now.Add(d))

			s, f := r.Inc(n)
			got := errorRate(s, f)
			if exp != got {
				t.Fatalf("expected %f, got %f", exp, got)
			}
		})
	}

	f("success", 0, 1, 0.0)
	f("failed", 0, -1, 0.5)
}

func TestResetErrors(t *testing.T) {
	perSecond := rate.NewErrors(time.Second)
	perSecond.Inc(-1)
	perSecond.Inc(1)
	perSecond.Reset()
	successes, failures := perSecond.Inc(0)
	exp := 0.0
	got := errorRate(successes, failures)
	if exp != got {
		t.Fatalf("expected %f, got %f", exp, got)
	}
}

func errorRate(successes, failures float64) float64 {
	num := failures
	den := failures + successes
	if den == 0 {
		return 0
	}

	return num / den
}
