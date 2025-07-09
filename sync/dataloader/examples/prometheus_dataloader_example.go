package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/dataloader"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func RunPrometheusDataLoaderExample() {
	totalRequests := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dataloader_total_requests",
		Help: "Total number of dataloader requests",
	})
	keysRequested := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dataloader_keys_requested",
		Help: "Total number of keys requested",
	})
	cacheHits := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dataloader_cache_hits",
		Help: "Total number of cache hits",
	})
	cacheMisses := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dataloader_cache_misses",
		Help: "Total number of cache misses",
	})
	batchCalls := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dataloader_batch_calls",
		Help: "Total number of batch calls",
	})
	errorCount := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dataloader_error_count",
		Help: "Total number of errors",
	})
	noResultCount := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dataloader_no_result_count",
		Help: "Total number of no result keys",
	})
	cacheSize := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dataloader_cache_size",
		Help: "Current cache size",
	})
	queueLength := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dataloader_queue_length",
		Help: "Current queue length",
	})

	prometheus.MustRegister(totalRequests, keysRequested, cacheHits, cacheMisses, batchCalls, errorCount, noResultCount, cacheSize, queueLength)

	metrics := &dataloader.PrometheusDataLoaderMetricsCollector{
		TotalRequests: totalRequests,
		KeysRequested: keysRequested,
		CacheHits:     cacheHits,
		CacheMisses:   cacheMisses,
		BatchCalls:    batchCalls,
		ErrorCount:    errorCount,
		NoResultCount: noResultCount,
		CacheSize:     cacheSize,
		QueueLength:   queueLength,
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		fmt.Println("Prometheus metrics at :2116/metrics")
		http.ListenAndServe(":2116", nil)
	}()

	batchFn := func(ctx context.Context, keys []string) (map[string]int, error) {
		m := make(map[string]int)
		for _, k := range keys {
			m[k] = len(k)
		}
		return m, nil
	}

	opts := &dataloader.Options[string, int]{
		BatchFn: batchFn,
	}
	loader := dataloader.New(context.Background(), opts, metrics)

	// Load some keys
	loader.Load("foo")
	loader.LoadMany([]string{"foo", "bar", "baz"})

	fmt.Println("DataLoader Prometheus example complete. Metrics at :2116/metrics")
	// Keep running to allow Prometheus scraping
	time.Sleep(5 * time.Second)
}
