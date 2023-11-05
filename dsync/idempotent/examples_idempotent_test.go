package idempotent_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/alextanhongpin/core/dsync/idempotent"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	redis "github.com/redis/go-redis/v9"
)

type Request struct {
	Age  int
	Name string
}

type Response struct {
	UserID int64
}

func ExampleIdempotent() {
	ctx := context.Background()
	stop := redistest.Init()
	defer stop()

	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	client.FlushAll(ctx)

	defer client.Close()

	idem := idempotent.New[Request, *Response](client, nil)

	req := Request{
		Age:  10,
		Name: "john",
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		h := func(ctx context.Context, req Request) (*Response, error) {
			fmt.Println("executing get user #1", req)
			time.Sleep(50 * time.Millisecond)
			return &Response{UserID: 10}, nil
		}

		v, err := idem.Do(ctx, "get-user", h, req)
		if err != nil {
			panic(err)
		}

		fmt.Printf("success #1: %+v\n", v)
	}()

	go func() {
		defer wg.Done()

		time.Sleep(50 * time.Millisecond)
		h := func(ctx context.Context, req Request) (*Response, error) {
			fmt.Println("executing get user #2", req)
			return &Response{UserID: 10}, nil
		}

		_, err := idem.Do(ctx, "get-user", h, req)
		if err == nil {
			fmt.Println(err)
			panic("want error, got nil")
		}
		fmt.Println("failed #2:", err)
		fmt.Println(errors.Is(err, idempotent.ErrRequestInFlight))
	}()

	wg.Wait()
	// Output:
	// executing get user #1 {10 john}
	// failed #2: idempotent: request in flight
	// true
	// success #1: &{UserID:10}
}
