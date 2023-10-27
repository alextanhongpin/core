package lock_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func TestLock(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})

	locker := lock.New(client)

	a := assert.New(t)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		time.Sleep(100 * time.Millisecond)
		err := locker.Lock(ctx, "hello", func(ctx context.Context) error {
			return nil
		})
		a.ErrorIs(err, lock.ErrLocked)
	}()

	go func() {
		defer wg.Done()

		err := locker.Lock(ctx, "hello", func(ctx context.Context) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		})
		a.Nil(err)
	}()

	wg.Wait()
}
