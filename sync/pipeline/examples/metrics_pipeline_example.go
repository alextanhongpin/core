package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/alextanhongpin/core/sync/pipeline"
)

// Example of a pipeline that uses the metrics interface (atomic implementation)
func main() {
	metrics := &pipeline.AtomicPipelineMetricsCollector{}
	metrics.SetStartTime(time.Now())

	in := make(chan int)
	go func() {
		for i := 0; i < 100; i++ {
			in <- i
		}
		close(in)
	}()

	// Simulate a pipeline with error and panic handling
	out := pipeline.Tap(in, func(v int) {
		metrics.IncProcessedCount()
		if v%10 == 0 {
			metrics.IncErrorCount()
		}
		if v == 42 {
			metrics.IncPanicCount()
		}
	})

	// Simulate throughput and error rate calculation
	start := time.Now()
	var processed, errors int64
	for v := range out {
		processed++
		if v%10 == 0 {
			errors++
		}
		// Simulate work
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	}
	dur := time.Since(start)
	metrics.SetDuration(dur)
	metrics.SetThroughputRate(float64(processed) / dur.Seconds())
	metrics.SetErrorRate(float64(errors) / float64(processed))

	fmt.Println("Pipeline metrics:")
	fmt.Println(metrics.GetMetrics().String())
}
