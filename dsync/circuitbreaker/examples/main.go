package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"github.com/redis/go-redis/v9"
)

var (
	requestTotal               = expvar.NewInt("request_total")
	circuitBrokenRequestsTotal = expvar.NewInt("circuit_broken_requests_total")
	stateChangedCount          = expvar.NewInt("state_changed_count")
	errorsTotal                = expvar.NewInt("errors_total")
)

// -k: keep-alive
// -c: concurrent users
// -n: number of requests
// NOTE: The URL must end with /
// ab -k -c 100 -n 20000 http://localhost:8080/?failure=1
func main() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	opt := circuitbreaker.NewOption()
	cb := circuitbreaker.New(client, opt)
	cb.OnStateChanged = func(ctx context.Context, from, to circuitbreaker.Status) {
		stateChangedCount.Add(1)
	}

	mux := http.NewServeMux()
	mux.Handle("/debug/vars", expvar.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requestTotal.Add(1)

		ctx := r.Context()

		rc := r.Clone(ctx)
		rc.URL.RawQuery = ""
		rc.URL.Fragment = ""

		err := cb.Do(ctx, rc.URL.String(), func() error {
			failure, _ := strconv.ParseBool(r.URL.Query().Get("failure"))
			if failure {
				return errors.New("bad request")
			}

			return nil
		})

		if errors.Is(err, circuitbreaker.ErrBrokenCircuit) {
			circuitBrokenRequestsTotal.Add(1)
			http.Error(w, "circuit open", http.StatusServiceUnavailable)
			return
		}

		if err != nil {
			errorsTotal.Add(1)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprint(w, "hello world")
	})

	log.Println("listening to port *:8080. press ctrl+c to cancel")
	http.ListenAndServe(":8080", mux)
}
