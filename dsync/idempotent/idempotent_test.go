package idempotent_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/alextanhongpin/core/dsync/idempotent"
	"github.com/alextanhongpin/dbtx/testing/redistest"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init()
	defer stop()

	m.Run()
}

func TestRedisStore(t *testing.T) {
	client := redistest.Client(t)
	t.Cleanup(func() {
		scanAll(client)
	})
	fn := func(ctx context.Context, req []byte) ([]byte, error) {
		scanAll(client)
		return []byte(`"world"`), nil
	}

	store := idempotent.NewRedisStore(client)
	res, shared, err := store.Do(ctx, t.Name(), fn, []byte(`"hello"`), time.Minute, time.Hour)
	is := assert.New(t)
	is.Nil(err)
	is.False(shared)
	is.Equal([]byte(`"world"`), res)

	res, shared, err = store.Do(ctx, t.Name(), fn, []byte(`"hello"`), time.Minute, time.Hour)
	is.Nil(err)
	is.True(shared)
	is.Equal([]byte(`"world"`), res)
}

func TestMakeHandler(t *testing.T) {
	fn := func(ctx context.Context, req string) (string, error) {
		return "world", nil
	}
	client := redistest.Client(t)
	t.Cleanup(func() {
		scanAll(client)
	})
	h := idempotent.NewHandler(client, fn, nil)

	res, shared, err := h.Handle(ctx, t.Name(), "hello")
	is := assert.New(t)
	is.Nil(err)
	is.False(shared)
	is.Equal("world", res)

	res, shared, err = h.Handle(ctx, t.Name(), "hello")
	is.Nil(err)
	is.True(shared)
	is.Equal("world", res)
}

func TestConcurrent(t *testing.T) {
	type Request struct {
		Msg string
	}
	type Response struct {
		Msg string
	}

	invoked := new(atomic.Int64)
	counter := new(atomic.Int64)
	inFlight := new(atomic.Int64)

	fn := func(ctx context.Context, req Request) (*Response, error) {
		invoked.Add(1)
		time.Sleep(100 * time.Millisecond)

		return &Response{
			Msg: strings.ToUpper(req.Msg),
		}, nil
	}

	client := redistest.Client(t)
	t.Cleanup(func() {
		scanAll(client)
	})
	h := idempotent.NewHandler(client, fn, nil)
	n := 10

	is := assert.New(t)

	var wg sync.WaitGroup
	wg.Add(3 * n)

	for range n {
		go func() {
			defer wg.Done()

			res, shared, err := h.Handle(ctx, t.Name(), Request{Msg: "hello"})
			is.Equal("HELLO", res.Msg)
			is.Nil(err)
			if shared {
				counter.Add(1)
			}
		}()

		go func() {
			defer wg.Done()

			time.Sleep(50 * time.Millisecond)
			h := idempotent.NewHandler(client, fn, nil)
			res, shared, err := h.Handle(ctx, t.Name(), Request{Msg: "hello"})
			if errors.Is(err, idempotent.ErrRequestInFlight) {
				inFlight.Add(1)
				return
			}
			is.Equal("HELLO", res.Msg)
			is.Nil(err)
			if shared {
				counter.Add(1)
			}
		}()

		go func() {
			defer wg.Done()
			time.Sleep(150 * time.Millisecond)

			h := idempotent.NewHandler(client, fn, nil)
			res, shared, err := h.Handle(ctx, t.Name(), Request{Msg: "hello"})
			if errors.Is(err, idempotent.ErrRequestInFlight) {
				inFlight.Add(1)
				return
			}
			is.Equal("HELLO", res.Msg)
			is.Nil(err)
			if shared {
				counter.Add(1)
			}
		}()
	}

	wg.Wait()
	is.Equal(int64(1), invoked.Load())
	is.Equal(int64(n*2-1), counter.Load())
	is.Equal(int64(n), inFlight.Load())
}

// TestExtendLock test the scenario where the callback function takes a longer
// time than the lock expiry, and the lock expired before the callback function
// completes.
// We expect the lock to be refresh periodically.
func TestExtendLock(t *testing.T) {
	client := redistest.Client(t)
	t.Cleanup(func() {
		scanAll(client)
	})

	fn := func(ctx context.Context, req string) (int, error) {
		// slow function
		time.Sleep(250 * time.Millisecond)
		return 42, nil
	}

	h := idempotent.NewHandler(client, fn, &idempotent.HandlerOptions{
		LockTTL: 100 * time.Millisecond,
		KeepTTL: 200 * time.Millisecond,
	})
	_, _, err := h.Handle(ctx, t.Name(), "world")
	if err != nil {
		t.Fatal(err)
	}
}

func scanAll(rdb *redis.Client) {
	// 2. Use Scan to safely find all keys without blocking Redis
	// Match "*" retrieves all keys. You can change this to "prefix:*" if needed.
	iter := rdb.Scan(ctx, 0, "*", 0).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		log.Fatalf("Failed to scan keys: %v", err)
	}

	if len(keys) == 0 {
		fmt.Println("No keys found in Redis.")
		return
	}

	// 3. Fetch all values efficiently using MGet (Multi-Get)
	values, err := rdb.MGet(ctx, keys...).Result()
	if err != nil {
		log.Fatalf("Failed to MGet values: %v", err)
	}

	// 4. Map keys to their respective values
	// Note: MGet returns nil for keys that don't exist or expired during execution
	fmt.Println("--- Key-Value Pairs ---")
	for i, key := range keys {
		val := values[i]
		if val == nil {
			fmt.Printf("%s: <nil> (key may have expired)\n", key)
		} else {
			fmt.Printf("%s: %v\n", key, val)
		}
	}
}
