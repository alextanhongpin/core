package metrics

import (
	"expvar"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Expvar counters updated by CounterHandler. They are published under these
// names when the package is imported.
var (
	StatusTotal   = expvar.NewMap("status_total")
	RequestsTotal = expvar.NewMap("requests_total")
)

// CounterHandler counts requests in total ("ALL"), by
// "<route pattern> - <status code>" in RequestsTotal, and by status code in
// StatusTotal. Expose them with:
//
//	mux.Handle("GET /debug/vars", expvar.Handler())
func CounterHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wr := &responseRecorder{ResponseWriter: w, code: http.StatusOK}
		h.ServeHTTP(wr, r)

		path := fmt.Sprintf("%s - %d", r.Pattern, wr.code)
		RequestsTotal.Add("ALL", 1)
		RequestsTotal.Add(path, 1)
		StatusTotal.Add(fmt.Sprint(wr.code), 1)
	})
}

// TrackerHandler records each request in tracker, keyed by
// "<route pattern> - <status code>" and the user returned by userFn. Record
// errors are logged to logger, or to slog.Default when logger is nil.
func TrackerHandler(h http.Handler, tracker *Tracker, userFn func(r *http.Request) string, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wr := &responseRecorder{ResponseWriter: w, code: http.StatusOK}
		h.ServeHTTP(wr, r)

		path := fmt.Sprintf("%s - %d", r.Pattern, wr.code)
		user := userFn(r)
		took := time.Since(start)
		err := tracker.Record(r.Context(), path, user, took)
		if err != nil {
			logger.Error(err.Error())
		}
	})
}

// TrackerStatsHandler writes tracker.Stats as plain text. The optional "at"
// query parameter (YYYY-MM-DD) selects the day; it defaults to today.
func TrackerStatsHandler(tracker *Tracker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := tracker.Now()
		if at := r.URL.Query().Get("at"); at != "" {
			t, err := time.Parse(time.DateOnly, at)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)

				return
			}
			now = t
		}

		stats, err := tracker.Stats(r.Context(), now)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		var sb strings.Builder
		for _, stat := range stats {
			fmt.Fprintf(&sb, "at: %s\n%s\n\n", now.Format(time.DateOnly), stat.String())
		}

		fmt.Fprint(w, sb.String())
	})
}
