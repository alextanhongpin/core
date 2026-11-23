# metrics

HTTP and operation metrics for Prometheus, an expvar request counter, and a
Redis-backed request tracker built on [`dsync/probs`](../dsync/probs).

```go
import "github.com/alextanhongpin/core/metrics"
```

## Prometheus

The collectors are package variables and are not registered. Register the
ones you use:

| Collector         | Type                  | Labels                              |
| ----------------- | --------------------- | ----------------------------------- |
| `InFlightGauge`   | gauge                 | –                                   |
| `RequestDuration` | histogram, seconds    | `method`, `path`, `status`, `version` |
| `ResponseSize`    | histogram, bytes      | –                                   |
| `RED`             | histogram, seconds    | `service`, `action`, `status`       |

```go
reg := prometheus.NewRegistry()
reg.MustRegister(metrics.InFlightGauge, metrics.RequestDuration, metrics.ResponseSize, metrics.RED)

mux := http.NewServeMux()
mux.HandleFunc("GET /users/{id}", getUser)

srv := http.NewServeMux()
srv.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
srv.Handle("/", metrics.RequestDurationHandler("v1.2.3", mux))
```

`RequestDurationHandler` updates all three HTTP collectors. The `path` label
is the matched `http.ServeMux` pattern without the method (`/users/{id}`), or
empty when nothing matched; raw URL paths are never used, so label values stay
bounded. It can wrap the mux or a handler registered on it, since the mux
sets the pattern on the request. A panicking handler is recorded with
status 500 and the panic is re-raised. `ResponseSize` is the number of body
bytes written.

### RED

```go
func processOrder(ctx context.Context, id string) error {
	red := metrics.NewRED("order_service", "process_order")
	defer red.Done()

	if err := validate(id); err != nil {
		red.Fail() // or red.SetStatus("invalid")
		return err
	}
	return nil
}
```

The status defaults to `metrics.OK`; `Fail` sets `metrics.Err`. A tracker is
not safe for concurrent use and `Done` must be called once. Every distinct
status is a separate series, so keep the set small.

## expvar

`CounterHandler` counts requests in the expvar maps `requests_total`
(`"ALL"` and `"<pattern> - <status>"`) and `status_total` (by status code).
Expose them with `mux.Handle("GET /debug/vars", expvar.Handler())`.

## Tracker

`Tracker` stores, per day and path, the request count (count-min sketch),
unique users (HyperLogLog) and p50/p90/p95 latency (t-digest), plus an
all-time top 10 of paths (top-k). It needs Redis with the RedisBloom module,
e.g. Redis Stack.

```go
tracker := metrics.NewTracker("api", redisClient)

userID := func(r *http.Request) string { return r.Header.Get("X-User-ID") }
mux.Handle("/", metrics.TrackerHandler(app, tracker, userID, slog.Default()))
mux.Handle("GET /admin/stats", metrics.TrackerStatsHandler(tracker)) // ?at=2024-01-31
```

The tracked path is `"<pattern> - <status>"`. `Stats(ctx, at)` returns the
statistics of the day of `at` for each all-time top path requested that day.
The day is taken from `Tracker.Now` (local time by default) when recording.

See [`cmd/main.go`](cmd/main.go) for a runnable server.
