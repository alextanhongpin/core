package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// The collectors below are not registered. Register the ones you use:
//
//	reg := prometheus.NewRegistry()
//	reg.MustRegister(metrics.InFlightGauge, metrics.RequestDuration, metrics.ResponseSize, metrics.RED)
var (
	// InFlightGauge counts requests currently served by RequestDurationHandler.
	InFlightGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "in_flight_requests",
			Help: "A gauge of requests currently being served by the wrapped handler.",
		},
	)

	// RequestDuration is partitioned by method, route pattern, status code
	// and version.
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status", "version"},
	)

	// ResponseSize records the number of response body bytes written.
	ResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "response_size_bytes",
			Help:    "A histogram of response sizes for requests.",
			Buckets: []float64{200, 500, 900, 1500},
		},
		[]string{},
	)

	// RED records the duration of operations tracked by REDTracker, in
	// seconds.
	RED = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "red_duration_seconds",
			Help:    "A histogram of operation latencies by service, action and status.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "action", "status"},
	)
)

// Status values used by REDTracker.
const (
	OK  string = "ok"
	Err string = "err"
)

// REDTracker records the rate, errors and duration of one operation into RED.
// It is not safe for concurrent use. Call Done exactly once.
type REDTracker struct {
	service string
	action  string
	status  string
	start   time.Time
}

// NewRED starts tracking an operation. The status defaults to OK.
func NewRED(service, action string) *REDTracker {
	return &REDTracker{
		service: service,
		action:  action,
		status:  OK,
		start:   time.Now(),
	}
}

// Done observes the elapsed time since NewRED.
func (r *REDTracker) Done() {
	RED.
		WithLabelValues(r.service, r.action, r.status).
		Observe(time.Since(r.start).Seconds())
}

// Fail sets the status to Err.
func (r *REDTracker) Fail() {
	r.status = Err
}

// SetStatus sets a custom status. Keep the set of values small; each one is
// a separate time series.
func (r *REDTracker) SetStatus(status string) {
	r.status = status
}

// RequestDurationHandler updates InFlightGauge, RequestDuration and
// ResponseSize for each request.
//
// The path label is the route pattern without the method (e.g. "/users/{id}")
// set by http.ServeMux, or empty when no pattern matched. It is never the raw
// URL path, which would create unbounded label values. A handler that panics
// is recorded with status 500 and the panic is re-raised.
func RequestDurationHandler(version string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		InFlightGauge.Inc()
		defer InFlightGauge.Dec()

		wr := &responseRecorder{ResponseWriter: w, code: http.StatusOK}
		start := time.Now()
		completed := false
		defer func() {
			code := wr.code
			if !completed {
				code = http.StatusInternalServerError
			}

			RequestDuration.
				WithLabelValues(r.Method, routePath(r.Pattern), strconv.Itoa(code), version).
				Observe(time.Since(start).Seconds())
			ResponseSize.WithLabelValues().Observe(float64(wr.size))
		}()

		next.ServeHTTP(wr, r)
		completed = true
	})
}

// routePath strips the optional method and host from a ServeMux pattern.
func routePath(pattern string) string {
	fields := strings.Fields(pattern)
	if len(fields) == 0 {
		return ""
	}

	return fields[len(fields)-1]
}

// responseRecorder captures the status code and number of bytes written.
type responseRecorder struct {
	http.ResponseWriter
	code        int
	size        int
	wroteHeader bool
}

func (w *responseRecorder) WriteHeader(code int) {
	if !w.wroteHeader && code >= 200 {
		// 1xx informational responses may precede the final status.
		w.code = code
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseRecorder) Write(b []byte) (int, error) {
	w.wroteHeader = true
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

// Flush keeps streaming handlers that assert http.Flusher working.
func (w *responseRecorder) Flush() {
	w.wroteHeader = true
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *responseRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
