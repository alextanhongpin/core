package singleflight_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/singleflight"
	"github.com/stretchr/testify/assert"
)

func TestSingleflight(t *testing.T) {
	is := assert.New(t)
	g := singleflight.New[int]()
	n := 10

	var wg sync.WaitGroup
	wg.Add(n)

	var exec atomic.Int64
	var share atomic.Int64
	ch := make(chan bool)
	for range n {
		go func() {
			defer wg.Done()

			<-ch

			res, shared, err := g.Do(context.Background(), "foo", func(ctx context.Context) (int, error) {
				exec.Add(1)

				// Add some sleep so that others will be waiting for the payload.
				time.Sleep(10 * time.Millisecond)

				return 42, nil
			})
			is.Nil(err)
			is.Equal(42, res)

			if shared {
				share.Add(1)
			}
		}()
	}
	close(ch)
	wg.Wait()

	is.Equal(int64(1), exec.Load())
	is.Equal(int64(9), share.Load())
}
