package rate_test

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"text/tabwriter"
	"time"

	"math/rand/v2"

	"github.com/alextanhongpin/core/sync/rate"
	"github.com/stretchr/testify/assert"
)

func ExampleLimiter() {
	r1 := rate.NewLimiter(10)
	r1.SuccessToken = 0.9

	r2 := rate.NewLimiter(10)
	r2.SuccessToken = 0.5

	r3 := rate.NewLimiter(10)
	r3.SuccessToken = 0.1

	var failure int
	allows := make([]int, 3)
	r := rand.New(rand.NewPCG(1, 2))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight|tabwriter.Debug)

	fmt.Fprintf(w, "%s\t%s\t%s\t\n", "0.9", "0.5", "0.1")
	for range 100 {
		a1, a2, a3 := r1.Allow(), r2.Allow(), r3.Allow()
		if a1 {
			allows[0]++
		}
		if a2 {
			allows[1]++
		}
		if a3 {
			allows[2]++
		}

		fmt.Fprintf(w, "%t\t%t\t%t\t\n", a1, a2, a3)
		if r.Float64() < 0.8 {
			failure++
			r1.Err()
			r2.Err()
			r3.Err()
		} else {
			r1.Ok()
			r2.Ok()
			r3.Ok()
		}
	}
	fmt.Fprintf(w, "%d\t%d\t%d\t\n", allows[0], allows[1], allows[2])
	fmt.Println("failure:", failure)
	fmt.Println("success token:")
	w.Flush()
	// Output:
	// failure: 68
	// success token:
	//    0.9|   0.5|   0.1|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true| false|
	//   true| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//   true|  true|  true|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//   true|  true|  true|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//   true|  true|  true|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//   true|  true|  true|
	//  false| false| false|
	//   true|  true|  true|
	//  false| false| false|
	//  false| false| false|
	//   true|  true|  true|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//   true|  true|  true|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true|  true| false|
	//   true|  true|  true|
	//   true|  true| false|
	//   true|  true|  true|
	//   true|  true| false|
	//   true| false| false|
	//   true| false| false|
	//   true|  true|  true|
	//   true| false| false|
	//   true|  true|  true|
	//   true| false| false|
	//   true| false| false|
	//   true|  true|  true|
	//   true| false| false|
	//   true|  true|  true|
	//   true| false| false|
	//   true|  true|  true|
	//   true| false| false|
	//  false| false| false|
	//   true|  true|  true|
	//   true|  true|  true|
	//   true| false| false|
	//   true|  true|  true|
	//   true| false| false|
	//   true|  true|  true|
	//   true| false| false|
	//   true|  true|  true|
	//   true| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//  false| false| false|
	//   true|  true|  true|
	//  false| false| false|
	//  false| false| false|
	//   true|  true|  true|
	//   true|  true|  true|
	//     59|    46|    42|
}

func TestLimit(t *testing.T) {
	badRequestErr := errors.New("bad request")

	t.Run("three consecutive errors", func(t *testing.T) {
		is := assert.New(t)
		limit := rate.NewLimiter(3)
		for range 3 {
			is.ErrorIs(limit.Do(func() error {
				return badRequestErr
			}), badRequestErr)
		}

		is.ErrorIs(limit.Do(func() error {
			return nil
		}), rate.ErrLimitExceeded)
		is.Equal(3, limit.Failure())
		is.Equal(0, limit.Success())
		is.Equal(3, limit.Total())
	})

	t.Run("two consecutive errors, one success, two consecutive errors", func(t *testing.T) {
		is := assert.New(t)
		limit := rate.NewLimiter(3)
		for range 2 {
			is.ErrorIs(limit.Do(func() error {
				return badRequestErr
			}), badRequestErr)
		}

		is.Nil(limit.Do(func() error {
			return nil
		}))

		for range 2 {
			is.ErrorIs(limit.Do(func() error {
				return badRequestErr
			}), badRequestErr)
		}

		is.ErrorIs(limit.Do(func() error {
			return nil
		}), rate.ErrLimitExceeded)

		is.Equal(4, limit.Failure())
		is.Equal(1, limit.Success())
		is.Equal(5, limit.Total())
	})

	t.Run("two consecutive errors, three successes, three consecutive errors", func(t *testing.T) {
		is := assert.New(t)
		limit := rate.NewLimiter(3)
		for range 2 {
			is.ErrorIs(limit.Do(func() error {
				return badRequestErr
			}), badRequestErr)
		}

		for range 3 {
			is.Nil(limit.Do(func() error {
				return nil
			}))
		}

		for range 3 {
			is.ErrorIs(limit.Do(func() error {
				return badRequestErr
			}), badRequestErr)
		}

		is.ErrorIs(limit.Do(func() error {
			return nil
		}), rate.ErrLimitExceeded)

		is.Equal(5, limit.Failure())
		is.Equal(3, limit.Success())
		is.Equal(8, limit.Total())
	})
}

func TestNewLimiterValidation(t *testing.T) {
	t.Run("zero limit panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for zero limit")
			}
		}()
		rate.NewLimiter(0)
	})

	t.Run("negative limit panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for negative limit")
			}
		}()
		rate.NewLimiter(-5)
	})

	t.Run("positive limit works", func(t *testing.T) {
		l := rate.NewLimiter(10)
		if l == nil {
			t.Error("expected valid limiter instance")
		}
	})
}

func TestLimiterConcurrency(t *testing.T) {
	t.Run("concurrent operations are safe", func(t *testing.T) {
		limiter := rate.NewLimiter(5)
		const numGoroutines = 10
		const numOperations = 100

		var wg sync.WaitGroup
		var successCount, errorCount int64

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					err := limiter.Do(func() error {
						// Simulate some work
						time.Sleep(time.Microsecond)
						// Randomly fail 30% of the time
						if j%3 == 0 {
							return errors.New("simulated error")
						}
						return nil
					})

					if err == nil {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}()
		}

		wg.Wait()

		// Verify that the counts make sense
		totalOps := successCount + errorCount
		if totalOps != numGoroutines*numOperations {
			t.Errorf("expected %d total operations, got %d", numGoroutines*numOperations, totalOps)
		}

		// The limiter should have blocked some operations due to failures
		if errorCount == 0 {
			t.Error("expected some operations to be blocked by rate limiter")
		}
	})
}
