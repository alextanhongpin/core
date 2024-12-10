package singleflight_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/singleflight"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestSingleflight(t *testing.T) {
	var (
		client  = redistest.New(t).Client()
		g       = singleflight.New(client)
		lockTTL = 10 * time.Second
		waitTTL = 10 * time.Second
	)

	t.Run("sync", func(t *testing.T) {
		key := t.Name()
		doOrWait, err := g.DoOrWait(ctx, key, func(ctx context.Context) error {
			return nil
		}, lockTTL, waitTTL)
		is := assert.New(t)
		is.Nil(err)
		is.True(doOrWait)

		doOrWait, err = g.DoOrWait(ctx, key, func(ctx context.Context) error {
			return nil
		}, lockTTL, waitTTL)
		is.Nil(err)
		is.True(doOrWait)
	})

	t.Run("concurrent", func(t *testing.T) {
		key := t.Name()
		is := assert.New(t)

		var did atomic.Int64
		var waited atomic.Int64

		n := 10

		ch := make(chan bool)
		var wg sync.WaitGroup
		wg.Add(n)

		// Shared instance depends on local lock.
		for range n / 2 {
			go func() {
				defer wg.Done()
				<-ch

				doOrWait, err := g.DoOrWait(ctx, key, func(ctx context.Context) error {
					did.Add(1)
					time.Sleep(100 * time.Millisecond)
					return nil
				}, lockTTL, waitTTL)
				is.Nil(err)
				if !doOrWait {
					waited.Add(1)
				}
			}()
		}

		// Individual instance waits for redis lock.
		for range n / 2 {
			go func() {
				defer wg.Done()
				<-ch

				g := singleflight.New(client)
				doOrWait, err := g.DoOrWait(ctx, key, func(ctx context.Context) error {
					did.Add(1)
					time.Sleep(100 * time.Millisecond)
					return nil
				}, lockTTL, waitTTL)
				is.Nil(err)
				if !doOrWait {
					waited.Add(1)
				}
			}()
		}
		close(ch)
		wg.Wait()
		is.Equal(int64(1), did.Load())
		is.Equal(int64(n-1), waited.Load())
	})
}
