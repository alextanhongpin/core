package idempotency_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/idempotency"
	"github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

type Request struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Response struct {
	Name string `json:"name"`
}

var SomeOperationKey = idempotency.Key("some-operation:%s")

func TestRequestReply(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	s := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer client.Close()

	do := func() {
		store := idempotency.NewRedisStore[*Response](client)
		handler := idempotency.NewRequestReply(store, idempotency.RequestReplyOption[Request, *Response]{
			LockTimeout:     5 * time.Second, // Default is 1 minute.
			RetentionPeriod: 1 * time.Minute, // Default is 24 hour.
			Handler: func(ctx context.Context, req Request) (*Response, error) {
				// Simulate critical section.
				time.Sleep(100 * time.Millisecond)

				return &Response{
					Name: "replied:" + req.Name,
				}, nil
			},
		})

		res, err := handler.Exec(ctx, SomeOperationKey.Format("xyz"), Request{
			ID:   "payout-123",
			Name: "foo",
		})
		if err != nil {
			assert.ErrorIs(err, idempotency.ErrRequestInFlight, err)
		} else {
			assert.Equal("replied:foo", res.Name)
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	race := make(chan bool)

	go func() {
		defer wg.Done()

		<-race
		do()
	}()

	go func() {
		defer wg.Done()

		<-race
		do()
	}()

	time.Sleep(100 * time.Millisecond)

	// Run concurrently.
	close(race)

	wg.Wait()

	s.CheckGet(t, "i9y:some-operation:xyz", `{"status":"success","request":"w93v/T90sFbhkHDcVqEfX1HWwArDAIFdNnppRNwjuKg=","response":{"name":"replied:foo"}}`)
}

func TestRequest(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	s := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer client.Close()

	do := func() {
		store := idempotency.NewRedisStore[any](client)
		handler := idempotency.NewRequest(store, idempotency.RequestOption[Request]{
			LockTimeout:     5 * time.Second, // Default is 1 minute.
			RetentionPeriod: 1 * time.Minute, // Default is 24 hour.
			Handler: func(ctx context.Context, req Request) error {
				// Simulate critical section.
				time.Sleep(100 * time.Millisecond)

				return nil
			},
		})

		err := handler.Exec(ctx, SomeOperationKey.Format("xyz"), Request{
			ID:   "payout-123",
			Name: "foo",
		})
		if err != nil {
			assert.ErrorIs(err, idempotency.ErrRequestInFlight, err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	race := make(chan bool)

	go func() {
		defer wg.Done()

		<-race
		do()
	}()

	go func() {
		defer wg.Done()

		<-race
		do()
	}()

	time.Sleep(100 * time.Millisecond)

	// Run concurrently.
	close(race)

	wg.Wait()

	s.CheckGet(t, "i9y:some-operation:xyz", `{"status":"success","request":"w93v/T90sFbhkHDcVqEfX1HWwArDAIFdNnppRNwjuKg="}`)
}
