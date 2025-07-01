package metrics

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alextanhongpin/core/http/httputil"
	"github.com/prometheus/client_golang/prometheus"
)

/*

	reg := prometheus.NewRegistry()
	// Install the default prometheus collectors.
	reg.MustRegister(collectors.NewGoCollector())
	// Install the custom metrics.
	reg.MustRegister(metrics.InFlightGauge, metrics.RequestDuration, metrics.ResponseSize)

	// ...
	metrics.InFlightGauge.Inc()
	metrics.InFlightGauge.Dec()
*/

var (
	InFlightGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "in_flight_requests",
			Help: "A gauge of requests currently being served by the wrapped handler.",
		},
	)

	// RequestDuration is partitioned by the HTTP method and handler. It uses custom
	// buckets based on the expected request duration.
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			// Prometheus recommended unit for duration is seconds (float).
			Name:    "request_duration_seconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status", "version"},
	)

	// ResponseSize has no labels, making it a zero-dimensional
	// ObserverVec.
	ResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "response_size_bytes",
			Help:    "A histogram of response sizes for requests.",
			Buckets: []float64{200, 500, 900, 1500},
		},
		[]string{},
	)

	RED = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "red",
			Help:    "RED metrics",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "action", "status"},
	)
)

const (
	OK  string = "ok"
	Err string = "err"
)

type REDTracker struct {
	service string
	action  string
	status  string
	Now     time.Time
}

func NewRED(service, action string) *REDTracker {
	return &REDTracker{
		service: service,
		action:  action,
		status:  OK,
		Now:     time.Now(),
	}
}

func (r *REDTracker) Done() {
	RED.
		WithLabelValues(r.service, r.action, r.status).
		Observe(float64(time.Since(r.Now).Milliseconds()))
}

func (r *REDTracker) Fail() {
	r.status = Err
}

func (r *REDTracker) SetStatus(status string) {
	r.status = status
}

func RequestDurationHandler(version string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wr := httputil.NewResponseWriterRecorder(w)

		defer func(start time.Time) {
			// Remove prefix http method, if any.
			method := r.Method
			path := tail(strings.Fields(r.Pattern))
			code := fmt.Sprintf("%d", wr.StatusCode())

			RequestDuration.
				WithLabelValues(method, path, code, version).
				Observe(float64(time.Since(start).Seconds()))
		}(time.Now())

		next.ServeHTTP(wr, r)
	})
}

func ObserveResponseSize(r *http.Request) int {
	if r == nil {
		return 0
	}
	size := computeApproximateRequestSize(r)
	ResponseSize.WithLabelValues().Observe(float64(size))
	return size
}

// Copied from prometheus source code.
func computeApproximateRequestSize(r *http.Request) int {
	if r == nil {
		return 0
	}

	s := 0
	if r.URL != nil {
		s += len(r.URL.String())
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}

func tail(vs []string) string {
	if len(vs) == 0 {
		return ""
	}

	return vs[len(vs)-1]
}
