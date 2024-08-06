// This package tests the ability to handle the idempotency for concurrent
// requests.
// We just fire 100 same requests, but multiple times. Due to idempotency,
// only 100 requests should be cached.
// We can also simulate calling with same idempotency key, but different
// payload. All of them should failed.
package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/alextanhongpin/core/dsync/idempotent"
	"github.com/redis/go-redis/v9"
)

var (
	requestTotal       = expvar.NewInt("request_total")
	successTotal       = expvar.NewInt("success_total")
	errRequestInFlight = expvar.NewInt("errors_request_in_flight_total")
	errRequestMismatch = expvar.NewInt("errors_request_mismatch_total")
	errKeyReleased     = expvar.NewInt("errors_key_released_total")
	errorsTotal        = expvar.NewInt("errors_total")
)

// -k: keep-alive
// -c: concurrent users
// -n: number of requests
// NOTE: The URL must end with /
// ab -k -c 100 -n 20000 http://localhost:8080/
func main() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	store := idempotent.NewRedisStore(client, &idempotent.Options{
		LockTTL: 1 * time.Second,
		KeepTTL: 1 * time.Hour,
	})

	fn := func(ctx context.Context, req int) (string, error) {
		// slow function
		sleep := time.Duration(rand.Intn(int(5 * time.Second)))
		time.Sleep(sleep)
		return fmt.Sprint(req), nil
	}
	h := idempotent.MakeHandler(store, fn)

	mux := http.NewServeMux()
	mux.Handle("/debug/vars", expvar.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mismatch, _ := strconv.ParseBool(r.URL.Query().Get("mismatch"))
		requestTotal.Add(1)
		n := rand.Intn(100)
		m := n
		if mismatch {
			m = n - 1
		}
		res, _, err := h.Do(r.Context(), fmt.Sprint(n), m)
		if err != nil {
			errorsTotal.Add(1)
			if errors.Is(err, idempotent.ErrRequestInFlight) {
				errRequestInFlight.Add(1)
			}
			if errors.Is(err, idempotent.ErrRequestMismatch) {
				errRequestMismatch.Add(1)
			}
			if errors.Is(err, idempotent.ErrLeaseInvalid) {
				errKeyReleased.Add(1)
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, res)
	})

	log.Println("listening to port *:8080. press ctrl+c to cancel")
	http.ListenAndServe(":8080", mux)
}
