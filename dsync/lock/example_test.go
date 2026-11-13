package lock_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	"github.com/redis/go-redis/v9"
)

func ExampleLocker_Do() {
	rdb := redis.NewClient(&redis.Options{Addr: redistest.Addr()})
	defer rdb.Close()

	locker, err := lock.New(lock.NewClient(rdb), lock.DefaultConfig())
	if err != nil {
		panic(err)
	}

	err = locker.Do(context.Background(), "example:reports:daily", func(ctx context.Context) error {
		// Only one process runs this at a time. Return early if ctx is canceled:
		// that means the lease was lost or the caller gave up.
		fmt.Println("rebuilding report")
		return nil
	})
	fmt.Println("err:", err)
	// Output:
	// rebuilding report
	// err: <nil>
}

// Fail fast instead of waiting when another process holds the key.
func ExampleNoRetry() {
	rdb := redis.NewClient(&redis.Options{Addr: redistest.Addr()})
	defer rdb.Close()

	ctx := context.Background()
	key := "example:no-retry"
	// Simulate another process holding the lease.
	rdb.Set(ctx, key, "someone-else", time.Second)
	defer rdb.Del(ctx, key)

	cfg := lock.DefaultConfig()
	cfg.Retry = lock.NoRetry()
	locker := lock.MustNew(lock.NewClient(rdb), cfg)

	err := locker.Do(ctx, key, func(ctx context.Context) error {
		fmt.Println("not printed")
		return nil
	})
	fmt.Println("busy:", errors.Is(err, lock.ErrLocked))
	// Output:
	// busy: true
}

// Func turns a request/response function into one that runs under a lock
// chosen per request.
func ExampleFunc() {
	rdb := redis.NewClient(&redis.Options{Addr: redistest.Addr()})
	defer rdb.Close()

	locker := lock.MustNew(lock.NewClient(rdb), lock.DefaultConfig())

	chargeOrder := func(ctx context.Context, orderID int) (string, error) {
		return fmt.Sprintf("charged order %d", orderID), nil
	}
	charge := lock.Func(chargeOrder, locker, func(_ context.Context, orderID int) (string, error) {
		return fmt.Sprintf("example:order:%d", orderID), nil
	})

	res, err := charge(context.Background(), 42)
	fmt.Println(res, err)
	// Output:
	// charged order 42 <nil>
}
