package metrics

import (
	"cmp"
	"context"
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"strings"
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

func TrackerHandler(h http.Handler, tracker *Tracker, userFn func(r *http.Request) string, timeFn func(time.Time) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wr := httputil.NewResponseWriterRecorder(w)
		h.ServeHTTP(wr, r)

		path := fmt.Sprintf("%s - %d", cmp.Or(r.Pattern, r.URL.Path), wr.StatusCode())
		tracker.Record(r.Context(), path, userFn(r), time.Since(start), timeFn(start))
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
}

func NewTracker(name string, client *redis.Client) *Tracker {
	return &Tracker{
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
}

func (t *Tracker) Record(ctx context.Context, path, userID string, duration time.Duration, when string) error {
	return errors.Join(
		t.countUnique(ctx, join(t.name, "hll", path, when), userID),
		t.countOccurences(ctx, join(t.name, "cms", when), path),
		t.rank(ctx, join(t.name, "top_k", when), path),
		t.recordLatency(ctx, join(t.name, "td", path, when), duration),
	)
}

func (t *Tracker) Stats(ctx context.Context, when string) ([]Stats, error) {
	paths, err := t.rankings(ctx, join(t.name, "top_k", when))
	if err != nil {
		return nil, err
	}

	stats := make([]Stats, len(paths))
	for i, path := range paths {
		unique, err := t.totalUnique(ctx, join(t.name, "hll", path, when))
		if err != nil {
			return nil, err
		}

		vals, err := t.latency(ctx, join(t.name, "td", path, when))
		if err != nil {
			return nil, err
		}

		total, err := t.totalOccurences(ctx, join(t.name, "cms", when), path)
		if err != nil {
			return nil, err
		}

		stats[i] = Stats{
			Path:   path,
			P50:    vals[0],
			P90:    vals[1],
			P95:    vals[2],
			Total:  total,
			Unique: unique,
		}
	}

	return stats, nil
}

func (t *Tracker) recordLatency(ctx context.Context, path string, duration time.Duration) error {
	_, err := t.td.Add(ctx, path, duration.Seconds())
	return err
}

func (t *Tracker) latency(ctx context.Context, path string) ([]float64, error) {
	return t.td.Quantile(ctx, path, 0.5, 0.9, 0.95)
}

func (t *Tracker) countOccurences(ctx context.Context, key, path string) error {
	_, err := t.cms.IncrBy(ctx, key, map[string]int64{
		path: 1,
	})
	return err
}

func (t *Tracker) totalOccurences(ctx context.Context, key, path string) (int64, error) {
	counts, err := t.cms.Query(ctx, key, path)
	if err != nil {
		return 0, err
	}

	return counts[0], nil
}

func (t *Tracker) countUnique(ctx context.Context, key, userID string) error {
	_, err := t.hll.Add(ctx, key, userID)
	return err
}

func (t *Tracker) totalUnique(ctx context.Context, key string) (int64, error) {
	return t.hll.Count(ctx, key)
}

func (t *Tracker) rank(ctx context.Context, key, path string) error {
	_, err := t.topK.Add(ctx, key, path)
	return err
}

func (t *Tracker) rankings(ctx context.Context, key string) ([]string, error) {
	return t.topK.List(ctx, key)
}

func (t *Tracker) hdmy(at time.Time) (h, d, m, y string) {
	h = at.Format("2006-01-02T15")
	d = at.Format("2006-01-02")
	m = at.Format("2006-01")
	y = at.Format("2006")

	return
}

type Stats struct {
	Path   string
	P50    float64
	P90    float64
	P95    float64
	Unique int64
	Total  int64
}

func (s *Stats) String() string {
	return fmt.Sprintf(`%s
unique/total: %d/%d
p50/p90/p95 (in seconds): %f, %f, %f`,
		s.Path,
		s.Unique, s.Total,
		s.P50, s.P90, s.P95,
	)
}

type Prefix string

func (p Prefix) Format(args ...any) string {
	return fmt.Sprintf(string(p), args...)
}

func join(s ...string) string {
	return strings.Join(s, ":")
}
