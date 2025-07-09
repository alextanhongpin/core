package main

import (
	"log"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/lock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newPrometheusLockMetrics() *lock.PrometheusLockMetricsCollector {
	activeLocks := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lock_active_locks",
		Help: "Number of active locks.",
	})
	totalLocks := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "lock_total_locks",
		Help: "Total locks created.",
	})
	lockAcquisitions := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "lock_acquisitions",
		Help: "Total lock acquisitions.",
	})
	lockContentions := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "lock_contentions",
		Help: "Number of lock contentions.",
	})
	totalWaitTime := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "lock_total_wait_time_ns",
		Help: "Total wait time for locks (nanoseconds).",
	})
	maxWaitTime := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lock_max_wait_time_ns",
		Help: "Maximum wait time observed (nanoseconds).",
	})

	prometheus.MustRegister(activeLocks, totalLocks, lockAcquisitions, lockContentions, totalWaitTime, maxWaitTime)

	return &lock.PrometheusLockMetricsCollector{
		ActiveLocks:      activeLocks,
		TotalLocks:       totalLocks,
		LockAcquisitions: lockAcquisitions,
		LockContentions:  lockContentions,
		TotalWaitTime:    totalWaitTime,
		MaxWaitTime:      maxWaitTime,
	}
}

func main() {
	metrics := newPrometheusLockMetrics()
	mgr := lock.NewWithOptions(lock.Options{}, metrics)

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		for {
			locker := mgr.Get("foo")
			locker.Lock()
			time.Sleep(100 * time.Millisecond)
			locker.Unlock()
			time.Sleep(200 * time.Millisecond)
		}
	}()

	log.Println("Prometheus metrics available at http://localhost:2112/metrics")
	log.Fatal(http.ListenAndServe(":2112", nil))
}
