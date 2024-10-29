package metrics

import (
	"cmp"
	"context"
	"expvar"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/core/http/httputil"
	redis "github.com/redis/go-redis/v9"
)

var (
	StatusTotal   = expvar.NewMap("status_total")
	RequestsTotal = expvar.NewMap("requests_total")
	ErrorsTotal   = expvar.NewMap("errors_total")
)

// CounterHandler tracks the success and error count.
// Install the expvar.Handler:
// mux.Handle("GET /debug/vars", expvar.Handler())
func CounterHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wr := httputil.NewResponseWriterRecorder(w)
		h.ServeHTTP(wr, r)

		path := fmt.Sprintf("%s - %d", cmp.Or(r.Pattern, r.URL.Path), wr.StatusCode())
		RequestsTotal.Add("ALL", 1)
		RequestsTotal.Add(path, 1)
		StatusTotal.Add(fmt.Sprint(wr.StatusCode()), 1)
	})
}

type UniqueCounter struct {
	Client *redis.Client
	Logger *slog.Logger
}

func (u *UniqueCounter) Handler(h http.Handler, key string, fn func(r *http.Request) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		val := fn(r)
		err := u.Client.PFAdd(r.Context(), key, val).Err()
		if err != nil {
			u.Logger.Error("failed to increment unique counter",
				slog.String("key", key),
				slog.String("val", val),
			)
		}
	})
}

// counters
// top-k
// rate-limits

// Of count-min-sketch, hyperloglog, top-k, t-digest
func RedisCounterHandler(h http.Handler, client *redis.Client, fn func(r *http.Request) string) http.Handler {
	// var bf BloomFilter
	//var g singleflight.Group
	ctx := context.Background()
	topK := probs.NewTopK(client)
	_, err := topK.Reserve(ctx, "top_k:requests_total", 10)
	if err != nil {
		panic(err)
	}
	td := probs.NewTDigest(client)

	cms := probs.NewCountMinSketch(ctx)
	_, err = cms.InitByProb(ctx, "count_min_sketch:requests_total", 0.01, 0.002)
	if err != nil {
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wr := httputil.NewResponseWriterRecorder(w)
		h.ServeHTTP(wr, r)

		path := fmt.Sprintf("%s - %d", cmp.Or(r.Pattern, r.URL.Path), wr.StatusCode())
		key := fn(r)
		ctx := r.Context()

		// TopK api calls.
		topK.Add(ctx, "top_k:requests_total", path)

		_, err = td.CreateWithCompression(ctx, "t_digest:requests_total:"+path, 100)
		if err != nil {
			panic(err)
		}
		defer func(start time.Time) {
			td.Add(ctx, "t_digest:requests_total:"+path, time.Since(start).Milliseconds())
		}(time.Now())
		_, err := cms.IncrBy(ctx, "count_min_sketch:requests_total", probs.Tuple[any, int]{
			K: path,
			V: 1,
		})
		if err != nil {
			panic(err)
		}

		// hyperloglog.add(path, user) - measure unique api calls
		// cms add(path, count) - track total api calls
		// t-digest add (path) - measure api latency
		// top-k add (path) - find top api calls

		if len(key) > 0 {
			// Check if exists.
			RequestsTotal.Add("ALL", 1)
			RequestsTotal.Add(path, 1)
			StatusTotal.Add(fmt.Sprint(wr.StatusCode()), 1)
		}
	})
}
