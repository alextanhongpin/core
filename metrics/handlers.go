package metrics

import (
	"cmp"
	"context"
	"errors"
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
func TrackerHandler(h http.Handler, tracker *Tracker, userFn func(r *http.Request) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wr := httputil.NewResponseWriterRecorder(w)
		h.ServeHTTP(wr, r)

		path := fmt.Sprintf("%s - %d", cmp.Or(r.Pattern, r.URL.Path), wr.StatusCode())
		tracker.Record(r.Context(), path, userFn(r), start)
	})
}

type Tracker struct {
	name string
	// t-digest add (path) - measure api latency
	td *probs.TDigest
	// cms add(path, count) - track total api calls
	cms *probs.CountMinSketch
	// hyperloglog.add(path, user) - measure unique api calls
	hll *probs.HyperLogLog
	// top-k add (path) - find top api calls
	topK *probs.TopK

	keys struct {
		td   string
		cms  string
		hll  string
		topK string
	}
}

func NewTracker(name string, client *redis.Client) *Tracker {
	t := &Tracker{
		name: name,
		// Track frequency of API calls.
		cms: probs.NewCountMinSketch(client),

		// Track unique page views by user
		hll: probs.NewHyperLogLog(client),

		// Track API latency
		td: probs.NewTDigest(client),

		// Track top-10 requests.
		topK: probs.NewTopK(client),
	}
	t.init()
	return t
}

func (t *Tracker) init() {
	t.keys.cms = fmt.Sprintf("%s:cms:api", t.name)
	t.keys.hll = fmt.Sprintf("%s:hll:api", t.name)
	t.keys.td = fmt.Sprintf("%s:td:api", t.name)
	t.keys.topK = fmt.Sprintf("%s:top_k:api", t.name)
}

func (t *Tracker) Record(ctx context.Context, path, userID string, start time.Time) error {
	return errors.Join(
		t.latency(ctx, path, start),
		t.countUnique(ctx, path),
		t.countOccurences(ctx, path, userID),
		t.top(ctx, path),
	)
}

func (t *Tracker) latency(ctx context.Context, path string, start time.Time) error {
	_, err := t.td.Add(ctx, t.keys.td+":"+path, time.Since(start).Seconds())
	return err
}

func (t *Tracker) countUnique(ctx context.Context, path string) error {
	_, err := t.cms.IncrBy(ctx, t.keys.cms, map[string]int64{
		path: 1,
	})
	return err
}

func (t *Tracker) countOccurences(ctx context.Context, page, userID string) error {
	now := time.Now()
	day := now.Format("2006-01-02")
	month := now.Format("2006-01")
	year := now.Format("2006")

	key := Prefix(t.keys.hll + ":%s:ts:%s")

	_, err1 := t.hll.Add(ctx, key.Format(page, day), userID)
	_, err2 := t.hll.Add(ctx, key.Format(page, month), userID)
	_, err3 := t.hll.Add(ctx, key.Format(page, year), userID)
	return errors.Join(err1, err2, err3)
}

func (t *Tracker) top(ctx context.Context, path string) error {
	_, err := t.topK.Add(ctx, t.keys.topK, path)
	return err
}

// TODO:
func (t *Tracker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stats, err := t.Stats(r.Context())
	if err != nil {
		panic(err)
	}
	_ = stats
}

func (t *Tracker) Stats(ctx context.Context) ([]Stats, error) {
	top10, err := t.topK.List(ctx, t.keys.topK)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	day := now.Format("2006-01-02")
	month := now.Format("2006-01")
	year := now.Format("2006")
	key := Prefix(t.keys.hll + ":%s:ts:%s")

	stats := make([]Stats, len(top10))
	for i, path := range top10 {
		daily, err := t.hll.Count(ctx, key.Format(path, day))
		if err != nil {
			return nil, err
		}
		monthly, err := t.hll.Count(ctx, key.Format(path, month))
		if err != nil {
			return nil, err
		}
		yearly, err := t.hll.Count(ctx, key.Format(path, year))
		if err != nil {
			return nil, err
		}
		vals, err := t.td.Quantile(ctx, t.keys.td+":"+path, 0.5, 0.9, 0.95)
		if err != nil {
			return nil, err
		}
		counts, err := t.cms.Query(ctx, t.keys.cms, path)
		if err != nil {
			return nil, err
		}

		stats[i] = Stats{
			Path:    path,
			P50:     vals[0],
			P90:     vals[1],
			P95:     vals[2],
			Total:   counts[0],
			Daily:   daily,
			Monthly: monthly,
			Yearly:  yearly,
		}
	}
	// List top10
	// for each top10 (or all)
	//   list page view (hll) day, weekly, monthly
	//   list latency (tdigest), 50, 95, 99%
	//   list number of calls (cms)

	return stats, nil
}

type Stats struct {
	Path    string
	P50     float64
	P90     float64
	P95     float64
	Daily   int64
	Monthly int64
	Yearly  int64
	Total   int64
}

func (s *Stats) String() string {
	return fmt.Sprintf(`%s
Requests total: %d
p50/p90/p95 (in seconds): %f, %f, %fs
d/m/y: %d/%d/%d`,
		s.Path,
		s.Total,
		s.P50, s.P90, s.P95,
		s.Daily, s.Monthly, s.Yearly,
	)
}

type Prefix string

func (p Prefix) Format(args ...any) string {
	return fmt.Sprintf(string(p), args...)
}
