package main

import (
	"context"
	"expvar"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alextanhongpin/core/metrics"
	redis "github.com/redis/go-redis/v9"
)

var client = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	tracker := metrics.NewTracker("hello", client)

	expvar.Publish("stats", expvar.Func(func() interface{} {
		stats, err := tracker.Stats(context.Background(), time.Now())
		if err != nil {
			logger.Error("failed to get stats", slog.String("err", err.Error()))
		}

		return stats
	}))

	userFn := func(r *http.Request) string {
		return r.RemoteAddr
	}

	mux := http.NewServeMux()
	mux.Handle("GET /debug/vars", expvar.Handler())
	mux.Handle("GET /", metrics.TrackerHandler(http.HandlerFunc(hello), tracker, userFn, logger))
	logger.Info("listening to port *:8080. press ctrl+c to cancel.")
	http.ListenAndServe(":8080", mux)
}

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello, %s", "world")
}
