package singleflight_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/singleflight"
	"github.com/stretchr/testify/assert"
)

func TestSingleflight(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		is := assert.New(t)
		g := singleflight.New[int]()
		n := 10

		var wg sync.WaitGroup
		wg.Add(n)

		var calls atomic.Int64
		var share atomic.Int64
		ch := make(chan bool)
		for range n {
			go func() {
				defer wg.Done()

				<-ch

				res, shared, err := g.Do(context.Background(), t.Name(), func(ctx context.Context) (int, error) {
					calls.Add(1)

					// Add some sleep so that others will be waiting for the payload.
					time.Sleep(10 * time.Millisecond)

					return 42, nil
				})
				is.NoError(err)
				is.Equal(42, res)

				if shared {
					share.Add(1)
				}
			}()
		}
		close(ch)
		wg.Wait()

		is.Equal(int64(1), calls.Load())
		is.Equal(int64(9), share.Load())
	})

	t.Run("failed", func(t *testing.T) {
		is := assert.New(t)
		g := singleflight.New[int]()
		n := 10

		var wg sync.WaitGroup
		wg.Add(n)

		var calls atomic.Int64
		var share atomic.Int64
		ch := make(chan bool)
		for range n {
			go func() {
				defer wg.Done()

				<-ch

				res, shared, err := g.Do(context.Background(), t.Name(), func(ctx context.Context) (int, error) {
					calls.Add(1)

					// Add some sleep so that others will be waiting for the payload.
					time.Sleep(10 * time.Millisecond)

					return 0, assert.AnError
				})
				is.ErrorIs(err, assert.AnError)
				is.Zero(res)

				if shared {
					share.Add(1)
				}
			}()
		}
		close(ch)
		wg.Wait()

		is.Equal(int64(1), calls.Load())
		is.Equal(int64(9), share.Load())
	})

	t.Run("expired", func(t *testing.T) {
		var calls atomic.Int64

		fn := func(ctx context.Context) (int, error) {
			calls.Add(1)
			return 42, nil
		}

		is := assert.New(t)
		g := singleflight.New[int]()
		for range 3 {
			res, shared, err := g.Do(context.Background(), t.Name(), fn)
			is.NoError(err)
			is.Equal(42, res)
			is.False(shared)
		}
		is.Equal(int64(3), calls.Load())
	})

	t.Run("parallel", func(t *testing.T) {
		is := assert.New(t)
		g := singleflight.New[int]()
		n := 10

		var wg sync.WaitGroup
		wg.Add(n)

		var calls atomic.Int64
		var share atomic.Int64
		ch := make(chan bool)
		for i := range n {
			go func(i int) {
				defer wg.Done()

				<-ch
				n := i % 5

				res, shared, err := g.Do(context.Background(), fmt.Sprintf("%s#%d", t.Name(), n), func(ctx context.Context) (int, error) {
					calls.Add(1)

					// Add some sleep so that others will be waiting for the payload.
					time.Sleep(10 * time.Millisecond)

					return n * 2, nil
				})
				is.NoError(err)
				is.Equal(n*2, res)
				if shared {
					share.Add(1)
				}
			}(i)
		}
		close(ch)
		wg.Wait()

		is.Equal(int64(5), calls.Load())
		is.Equal(int64(5), share.Load())
	})
}
