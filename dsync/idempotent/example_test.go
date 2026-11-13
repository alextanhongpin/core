package idempotent_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/alextanhongpin/core/dsync/idempotent"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	redis "github.com/redis/go-redis/v9"
)

// The first call runs the function; repeating the same key and request
// replays the stored response without running it again.
func ExampleIdempotent_HandlerFunc() {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: redistest.Addr()})
	defer rdb.Close()

	idem, err := idempotent.NewWithRedis(rdb)
	if err != nil {
		panic(err)
	}

	greet, err := idem.HandlerFunc(func(ctx context.Context, name string) (string, error) {
		fmt.Println("running task")
		return "Hello, " + name, nil
	}, idempotent.HandlerConfig{}) // zero value: LockTTL=10s, KeepTTL=24h
	if err != nil {
		panic(err)
	}

	key := "example:greet:req-1"
	rdb.Del(ctx, key) // start clean so the example is repeatable

	for range 2 {
		res, shared, err := greet.Do(ctx, key, "Ada")
		if err != nil {
			panic(err)
		}
		fmt.Println(res, "shared:", shared)
	}
	// Output:
	// running task
	// Hello, Ada shared: false
	// Hello, Ada shared: true
}

type ChargeRequest struct {
	AccountID string `json:"account_id"`
	Cents     int64  `json:"cents"`
}

type Receipt struct {
	ID string `json:"id"`
}

type chargeTask struct{ next int }

func (t *chargeTask) Do(ctx context.Context, req ChargeRequest) (Receipt, error) {
	t.next++
	return Receipt{ID: fmt.Sprintf("rcpt-%d", t.next)}, nil
}

// Handler accepts any type implementing [idempotent.Task]. Reusing a key for a
// different request is reported as [idempotent.ErrRequestMismatch].
func ExampleIdempotent_Handler() {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: redistest.Addr()})
	defer rdb.Close()

	charge := idempotent.MustNewWithRedis(rdb).MustHandler(&chargeTask{}, idempotent.HandlerConfig{})

	key := "example:tenant-1:charge:abc"
	rdb.Del(ctx, key) // start clean so the example is repeatable

	r1, _, _ := charge.Do(ctx, key, ChargeRequest{AccountID: "acc-1", Cents: 500})
	r2, shared, _ := charge.Do(ctx, key, ChargeRequest{AccountID: "acc-1", Cents: 500})
	fmt.Println(r1.ID, r2.ID, shared)

	_, _, err := charge.Do(ctx, key, ChargeRequest{AccountID: "acc-1", Cents: 999})
	fmt.Println(errors.Is(err, idempotent.ErrRequestMismatch))
	// Output:
	// rcpt-1 rcpt-1 true
	// true
}

// Map the package's errors to HTTP status codes.
func Example_httpStatus() {
	status := func(err error) int {
		switch {
		case err == nil:
			return http.StatusOK
		case errors.Is(err, idempotent.ErrEmptyKey):
			return http.StatusBadRequest // missing Idempotency-Key header
		case errors.Is(err, idempotent.ErrRequestMismatch):
			return http.StatusUnprocessableEntity // key reused with another body
		case errors.Is(err, idempotent.ErrRequestInFlight):
			return http.StatusConflict // still running; retry later
		default:
			return http.StatusInternalServerError // task, Redis or lease error
		}
	}

	fmt.Println(status(nil))
	fmt.Println(status(idempotent.ErrRequestMismatch))
	fmt.Println(status(idempotent.ErrRequestInFlight))
	// Output:
	// 200
	// 422
	// 409
}
