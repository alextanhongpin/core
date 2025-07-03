package rate_test

import (
	"fmt"
	"math"
	"os"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
	"github.com/stretchr/testify/assert"
)

func ExampleRate() {
	testRate("constant 100ms", func(i int) time.Duration {
		return time.Duration(i) * 100 * time.Millisecond
	})
	testRate("constant 1s", func(i int) time.Duration {
		return time.Duration(i) * time.Second
	})
	testRate("constant 10s", func(i int) time.Duration {
		return time.Duration(i) * 10 * time.Second
	})
	testRate("constant 1m", func(i int) time.Duration {
		return time.Duration(i) * time.Minute
	})
	testRate("exponential", func(i int) time.Duration {
		return time.Duration(math.Pow(2, float64(i))) * 1 * time.Millisecond
	})
	// Output:
	// constant 100ms
	//     ms|    s|     m|     h|
	//     0s| 1.00|  1.00|  1.00|
	//  100ms| 1.90|  2.00|  2.00|
	//  200ms| 2.71|  3.00|  3.00|
	//  300ms| 3.44|  3.99|  4.00|
	//  400ms| 4.10|  4.98|  5.00|
	//  500ms| 4.69|  5.98|  6.00|
	//  600ms| 5.22|  6.97|  7.00|
	//  700ms| 5.70|  7.95|  8.00|
	//  800ms| 6.13|  8.94|  9.00|
	//  900ms| 6.51|  9.93| 10.00|
	//     1s| 6.86| 10.91| 11.00|
	//   1.1s| 7.18| 11.89| 12.00|
	//   1.2s| 7.46| 12.87| 13.00|
	//   1.3s| 7.71| 13.85| 14.00|
	//   1.4s| 7.94| 14.83| 15.00|
	//   1.5s| 8.15| 15.80| 16.00|
	//   1.6s| 8.33| 16.78| 17.00|
	//   1.7s| 8.50| 17.75| 18.00|
	//   1.8s| 8.65| 18.72| 19.00|
	//   1.9s| 8.78| 19.69| 19.99|
	//
	// constant 1s
	//   ms|    s|     m|     h|
	//   0s| 1.00|  1.00|  1.00|
	//   1s| 1.00|  1.98|  2.00|
	//   2s| 1.00|  2.95|  3.00|
	//   3s| 1.00|  3.90|  4.00|
	//   4s| 1.00|  4.84|  5.00|
	//   5s| 1.00|  5.76|  6.00|
	//   6s| 1.00|  6.66|  6.99|
	//   7s| 1.00|  7.55|  7.99|
	//   8s| 1.00|  8.42|  8.99|
	//   9s| 1.00|  9.28|  9.99|
	//  10s| 1.00| 10.13| 10.98|
	//  11s| 1.00| 10.96| 11.98|
	//  12s| 1.00| 11.78| 12.98|
	//  13s| 1.00| 12.58| 13.97|
	//  14s| 1.00| 13.37| 14.97|
	//  15s| 1.00| 14.15| 15.97|
	//  16s| 1.00| 14.91| 16.96|
	//  17s| 1.00| 15.66| 17.96|
	//  18s| 1.00| 16.40| 18.95|
	//  19s| 1.00| 17.13| 19.95|
	//
	// constant 10s
	//     ms|    s|    m|     h|
	//     0s| 1.00| 1.00|  1.00|
	//    10s| 1.00| 1.83|  2.00|
	//    20s| 1.00| 2.53|  2.99|
	//    30s| 1.00| 3.11|  3.98|
	//    40s| 1.00| 3.59|  4.97|
	//    50s| 1.00| 3.99|  5.96|
	//   1m0s| 1.00| 4.33|  6.94|
	//  1m10s| 1.00| 4.60|  7.92|
	//  1m20s| 1.00| 4.84|  8.90|
	//  1m30s| 1.00| 5.03|  9.88|
	//  1m40s| 1.00| 5.19| 10.85|
	//  1m50s| 1.00| 5.33| 11.82|
	//   2m0s| 1.00| 5.44| 12.79|
	//  2m10s| 1.00| 5.53| 13.75|
	//  2m20s| 1.00| 5.61| 14.71|
	//  2m30s| 1.00| 5.68| 15.67|
	//  2m40s| 1.00| 5.73| 16.63|
	//  2m50s| 1.00| 5.77| 17.58|
	//   3m0s| 1.00| 5.81| 18.53|
	//  3m10s| 1.00| 5.84| 19.48|
	//
	// constant 1m
	//     ms|    s|    m|     h|
	//     0s| 1.00| 1.00|  1.00|
	//   1m0s| 1.00| 1.00|  1.98|
	//   2m0s| 1.00| 1.00|  2.95|
	//   3m0s| 1.00| 1.00|  3.90|
	//   4m0s| 1.00| 1.00|  4.84|
	//   5m0s| 1.00| 1.00|  5.76|
	//   6m0s| 1.00| 1.00|  6.66|
	//   7m0s| 1.00| 1.00|  7.55|
	//   8m0s| 1.00| 1.00|  8.42|
	//   9m0s| 1.00| 1.00|  9.28|
	//  10m0s| 1.00| 1.00| 10.13|
	//  11m0s| 1.00| 1.00| 10.96|
	//  12m0s| 1.00| 1.00| 11.78|
	//  13m0s| 1.00| 1.00| 12.58|
	//  14m0s| 1.00| 1.00| 13.37|
	//  15m0s| 1.00| 1.00| 14.15|
	//  16m0s| 1.00| 1.00| 14.91|
	//  17m0s| 1.00| 1.00| 15.66|
	//  18m0s| 1.00| 1.00| 16.40|
	//  19m0s| 1.00| 1.00| 17.13|
	//
	// exponential
	//         ms|    s|     m|     h|
	//        1ms| 1.00|  1.00|  1.00|
	//        2ms| 2.00|  2.00|  2.00|
	//        4ms| 3.00|  3.00|  3.00|
	//        8ms| 3.98|  4.00|  4.00|
	//       16ms| 4.95|  5.00|  5.00|
	//       32ms| 5.87|  6.00|  6.00|
	//       64ms| 6.68|  6.99|  7.00|
	//      128ms| 7.26|  7.99|  8.00|
	//      256ms| 7.33|  8.97|  9.00|
	//      512ms| 6.45|  9.93| 10.00|
	//     1.024s| 4.15| 10.85| 11.00|
	//     2.048s| 1.00| 11.66| 11.99|
	//     4.096s| 1.00| 12.26| 12.99|
	//     8.192s| 1.00| 12.43| 13.97|
	//    16.384s| 1.00| 11.73| 14.94|
	//    32.768s| 1.00|  9.53| 15.87|
	//   1m5.536s| 1.00|  5.32| 16.73|
	//  2m11.072s| 1.00|  1.00| 17.42|
	//  4m22.144s| 1.00|  1.00| 17.79|
	//  8m44.288s| 1.00|  1.00| 17.49|
}

func testRate(name string, fn func(i int) time.Duration) {
	r1 := rate.NewRate(time.Second)
	r2 := rate.NewRate(time.Minute)
	r3 := rate.NewRate(time.Hour)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", "ms", "s", "m", "h")

	now := time.Now()
	for i := range 20 {
		sleep := fn(i)
		r1.Now = func() time.Time { return now.Add(sleep) }
		r2.Now = func() time.Time { return now.Add(sleep) }
		r3.Now = func() time.Time { return now.Add(sleep) }
		fmt.Fprintf(w, "%s\t%0.2f\t%.2f\t%.2f\t\n", sleep, r1.Inc(), r2.Inc(), r3.Inc())
	}
	fmt.Println(name)
	w.Flush()
	fmt.Println()
}

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

			r.Success().Add(success)
			r.Failure().Add(failure)

			is := assert.New(t)
			is.Equal(want, r.Rate().Ratio())
			is.Equal(success, r.Rate().Success(), "success")
			is.Equal(failure, r.Rate().Failure(), "failure")
		})
	}

	f("success", 0, 1, 0, 0.0)
	f("failed", 0, 0, 1, 1)
	f("failed", 0, 1, 1, 0.5)
}

func TestResetErrors(t *testing.T) {
	ps := rate.NewErrors(time.Second)
	ps.Success().Inc()
	ps.Failure().Inc()
	ps.Reset()

	is := assert.New(t)
	is.Equal(0.0, ps.Rate().Ratio())
	is.Equal(0.0, ps.Rate().Success())
	is.Equal(0.0, ps.Rate().Failure())
}

func TestNewRateValidation(t *testing.T) {
	t.Run("zero period panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for zero period")
			}
		}()
		rate.NewRate(0)
	})

	t.Run("negative period panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for negative period")
			}
		}()
		rate.NewRate(-time.Second)
	})

	t.Run("positive period works", func(t *testing.T) {
		r := rate.NewRate(time.Second)
		if r == nil {
			t.Error("expected valid rate instance")
		}
	})
}
