package ratelimit_test

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	redis "github.com/redis/go-redis/v9"
)

// The examples run against the Redis container started in TestMain.
// In your code, use your own address, e.g. "localhost:6379".
func newExampleClient(keys ...string) *redis.Client {
	client := redis.NewClient(&redis.Options{Addr: redistest.Addr()})
	ctx := context.Background()
	if err := ratelimit.Setup(ctx, client); err != nil {
		panic(err)
	}
	client.Del(ctx, keys...) // Start from a clean state.
	return client
}

func ExampleNewFixedWindow() {
	ctx := context.Background()
	client := newExampleClient("example:fixed:user:1")
	defer client.Close()

	// At most 3 requests per user per minute.
	limiter, err := ratelimit.NewFixedWindow(client, ratelimit.Config{Limit: 3, Period: time.Minute})
	if err != nil {
		panic(err)
	}

	for i := 1; i <= 4; i++ {
		ok, err := limiter.Allow(ctx, "example:fixed:user:1")
		if err != nil {
			panic(err)
		}
		fmt.Printf("request %d allowed: %t\n", i, ok)
	}
	// Output:
	// request 1 allowed: true
	// request 2 allowed: true
	// request 3 allowed: true
	// request 4 allowed: false
}

func ExampleNewGCRA() {
	ctx := context.Background()
	client := newExampleClient("example:gcra:user:1")
	defer client.Close()

	// One request per second on average; an idle user may send 2 extra
	// requests at once (3 in total).
	limiter, err := ratelimit.NewGCRA(client, ratelimit.Config{Limit: 1, Period: time.Second, Burst: 2})
	if err != nil {
		panic(err)
	}

	for i := 1; i <= 4; i++ {
		res, err := limiter.Limit(ctx, "example:gcra:user:1")
		if err != nil {
			panic(err)
		}
		fmt.Printf("request %d allowed: %t, remaining: %d\n", i, res.Allow, res.Remaining)
	}
	// Output:
	// request 1 allowed: true, remaining: 2
	// request 2 allowed: true, remaining: 1
	// request 3 allowed: true, remaining: 0
	// request 4 allowed: false, remaining: 0
}

// RateLimit is an HTTP middleware that limits each API key and answers
// 429 Too Many Requests with a Retry-After header when the quota is used up.
func RateLimit(limiter ratelimit.RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := "ratelimit:api:" + r.Header.Get("X-API-Key")
		res, err := limiter.Limit(r.Context(), key)
		if err != nil {
			// Redis is unavailable. Here we fail closed; you may prefer to
			// let the request through instead.
			http.Error(w, "rate limiter unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		if !res.Allow {
			// Retry-After is in whole seconds; round up so clients don't retry too early.
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(res.RetryAfter.Seconds()))))
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func Example_httpMiddleware() {
	client := newExampleClient("ratelimit:api:secret")
	defer client.Close()

	limiter := ratelimit.MustNewFixedWindow(client, ratelimit.Config{Limit: 2, Period: time.Minute})
	hello := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	})
	handler := RateLimit(limiter, hello)

	for range 3 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-API-Key", "secret")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		fmt.Printf("%d Retry-After=%q\n", rec.Code, rec.Header().Get("Retry-After"))
	}
	// Output:
	// 200 Retry-After=""
	// 200 Retry-After=""
	// 429 Retry-After="60"
}
