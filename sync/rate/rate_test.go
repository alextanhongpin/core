package rate_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
	"github.com/stretchr/testify/assert"
)

func TestRate(t *testing.T) {
	now := time.Now()
	period := 5 * time.Second

	r := rate.NewRate(period)
	f := func(name string, d time.Duration, want float64) {
		t.Run(name, func(t *testing.T) {
			r.Now = func() time.Time { return now.Add(d) }
			got := r.Inc()
			if want != got {
				t.Fatalf("expected %f, got %f", want, got)
			}
		})
	}

	f("first", 0, 1)
	f("second", 1*time.Second, 1.8)
	f("third", 2*time.Second, 2.44)
	f("fourth", 3*time.Second, 2.952)
	f("fifth", 4*time.Second, 3.3616)
	f("reset", 5*time.Second, 3.68928)
}

func TestResetRate(t *testing.T) {
	ps := rate.NewRate(time.Second)
	ps.Inc()
	ps.Reset()
	got := ps.Count()
	want := float64(0.0)
	if want != got {
		t.Fatalf("expected %f, got %f,", want, got)
	}
}

func TestErrors(t *testing.T) {
	now := time.Now()
	period := 5 * time.Second

	f := func(name string, d time.Duration, success, failure float64, want float64) {
		t.Run(name, func(t *testing.T) {
			r := rate.NewErrors(period)
			r.SetNow(func() time.Time {
				return now.Add(d)
			})

			r.AddSuccess(success)
			r.AddFailure(failure)

			is := assert.New(t)
			is.Equal(want, r.Result().ErrorRate())
			is.Equal(success, r.Result().Success, "success")
			is.Equal(failure, r.Result().Failure, "failure")
		})
	}

	f("success", 0, 1, 0, 0.0)
	f("failed", 0, 0, 1, 1)
	f("failed", 0, 1, 1, 0.5)
}

func TestResetErrors(t *testing.T) {
	ps := rate.NewErrors(time.Second)
	ps.IncSuccess()
	ps.IncFailure()
	ps.Reset()

	is := assert.New(t)
	is.Equal(0.0, ps.Result().ErrorRate())
	is.Equal(0.0, ps.Result().Success)
	is.Equal(0.0, ps.Result().Failure)
}
