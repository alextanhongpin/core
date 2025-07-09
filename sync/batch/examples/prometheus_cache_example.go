package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/batch"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func RunPrometheusCacheExample() {
	// Create Prometheus metrics
	gets := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cache_gets_total",
		Help: "Total number of cache get operations",
	})
	sets := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cache_sets_total",
		Help: "Total number of cache set operations",
	})
	hits := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cache_hits_total",
		Help: "Total number of cache hits",
	})
	misses := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cache_misses_total",
		Help: "Total number of cache misses",
	})
	evictions := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cache_evictions_total",
		Help: "Total number of cache evictions",
	})
	size := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cache_size",
		Help: "Current cache size",
	})

	prometheus.MustRegister(gets, sets, hits, misses, evictions, size)

	metrics := &batch.PrometheusCacheMetricsCollector{
		Gets:      gets,
		Sets:      sets,
		Hits:      hits,
		Misses:    misses,
		Evictions: evictions,
		Size:      size,
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		fmt.Println("Prometheus metrics at :2113/metrics")
		http.ListenAndServe(":2113", nil)
	}()

	cache := batch.NewCache[string, int](metrics)
	ctx := context.Background()

	// Store values
	cache.StoreMany(ctx, map[string]int{"a": 1, "b": 2}, 5*time.Second)
	// Load values
	cache.LoadMany(ctx, "a", "b", "c")
	// Cleanup expired (simulate after some time)
	time.Sleep(6 * time.Second)
	cache.CleanupExpired()

	fmt.Println("Cache Prometheus example complete. Metrics at :2113/metrics")
}
