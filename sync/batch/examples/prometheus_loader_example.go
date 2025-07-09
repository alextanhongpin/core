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

func RunPrometheusLoaderExample() {
	// Create Prometheus metrics
	cacheHits := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "loader_cache_hits_total",
		Help: "Total number of loader cache hits",
	})
	cacheMisses := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "loader_cache_misses_total",
		Help: "Total number of loader cache misses",
	})
	batchCalls := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "loader_batch_calls_total",
		Help: "Total number of loader batch calls",
	})
	totalKeys := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "loader_total_keys_total",
		Help: "Total number of loader keys requested",
	})
	errorCount := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "loader_error_count_total",
		Help: "Total number of loader errors",
	})

	prometheus.MustRegister(cacheHits, cacheMisses, batchCalls, totalKeys, errorCount)

	metrics := &batch.PrometheusLoaderMetricsCollector{
		CacheHits:   cacheHits,
		CacheMisses: cacheMisses,
		BatchCalls:  batchCalls,
		TotalKeys:   totalKeys,
		ErrorCount:  errorCount,
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		fmt.Println("Prometheus metrics at :2114/metrics")
		http.ListenAndServe(":2114", nil)
	}()

	// Simple batch function
	batchFn := func(keys []string) (map[string]int, error) {
		m := make(map[string]int)
		for _, k := range keys {
			m[k] = len(k)
		}
		return m, nil
	}

	cache := batch.NewCache[string, *batch.Result[int]]()
	opts := &batch.LoaderOptions[string, int]{
		Cache:   cache,
		BatchFn: batchFn,
		TTL:     5 * time.Second,
	}
	loader := batch.NewLoader(opts, metrics)
	ctx := context.Background()

	// Load some keys
	loader.LoadMany(ctx, []string{"foo", "bar", "baz"})
	loader.LoadMany(ctx, []string{"foo", "baz", "qux"})

	fmt.Println("Loader Prometheus example complete. Metrics at :2114/metrics")
}
