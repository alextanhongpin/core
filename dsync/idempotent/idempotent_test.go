package idempotent

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/alextanhongpin/core/storage/redis/redistest"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func TestRedisStore(t *testing.T) {
	fn := func(ctx context.Context, req []byte) ([]byte, error) {
		return []byte("world"), nil
	}

	store := NewRedisStore(newClient(t), nil)
	res, shared, err := store.Do(ctx, t.Name(), fn, []byte("hello"))
	is := assert.New(t)
	is.Nil(err)
	is.False(shared)
	is.Equal([]byte("world"), res)

	res, shared, err = store.Do(ctx, t.Name(), fn, []byte("hello"))
	is.Nil(err)
	is.True(shared)
	is.Equal([]byte("world"), res)
}

func TestMakeHandler(t *testing.T) {
	store := NewRedisStore(newClient(t), nil)
	h := MakeHandler(store, func(ctx context.Context, req string) (string, error) {
		return "world", nil
	})

	res, shared, err := h.Do(ctx, t.Name(), "hello")
	is := assert.New(t)
	is.Nil(err)
	is.False(shared)
	is.Equal("world", res)

	res, shared, err = h.Do(ctx, t.Name(), "hello")
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

	client := newClient(t)
	store := NewRedisStore(client, nil)
	h := MakeHandler(store, fn)
	n := 10

	is := assert.New(t)

	var wg sync.WaitGroup
	wg.Add(3 * n)

	for range n {
		go func() {
			defer wg.Done()

			res, shared, err := h.Do(ctx, t.Name(), Request{Msg: "hello"})
			if errors.Is(err, ErrRequestInFlight) {
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

			h := MakeHandler(store, fn)
			res, shared, err := h.Do(ctx, t.Name(), Request{Msg: "hello"})
			if errors.Is(err, ErrRequestInFlight) {
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

			h := MakeHandler(store, fn)
			res, shared, err := h.Do(ctx, t.Name(), Request{Msg: "hello"})
			if errors.Is(err, ErrRequestInFlight) {
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
	is.Equal(int64(n), counter.Load())
	is.Equal(int64(n*2-1), inFlight.Load())
}

// TestExtendLock test the scenario where the callback function takes a longer
// time than the lock expiry, and the lock expired before the callback function
// completes.
// We expect the lock to be refresh periodically.
func TestExtendLock(t *testing.T) {
	client := newClient(t)

	fn := func(ctx context.Context, req string) (int, error) {
		// slow function
		time.Sleep(250 * time.Millisecond)
		return 42, nil
	}

	store := NewRedisStore(client, nil)
	opts := []Option{WithKeepTTL(200 * time.Millisecond), WithLockTTL(100 * time.Millisecond)}
	h := MakeHandler(store, fn, opts...)
	_, _, err := h.Do(ctx, t.Name(), "world")
	if err != nil {
		t.Fatal(err)
	}
}

func newClient(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})

	t.Helper()
	t.Cleanup(func() {
		client.FlushAll(ctx).Err()
		client.Close()
	})

	return client
}
