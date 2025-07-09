package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func RunPrometheusCircuitBreakerExample() {
	totalRequests := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cb_total_requests",
		Help: "Total number of circuit breaker requests",
	})
	successfulRequests := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cb_successful_requests",
		Help: "Number of successful requests",
	})
	failedRequests := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cb_failed_requests",
		Help: "Number of failed requests",
	})
	rejectedRequests := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cb_rejected_requests",
		Help: "Number of requests rejected due to open circuit",
	})
	stateTransitions := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cb_state_transitions",
		Help: "Number of state transitions",
	})
	currentState := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cb_current_state",
		Help: "Current state of the circuit breaker",
	}, []string{"state"})

	prometheus.MustRegister(totalRequests, successfulRequests, failedRequests, rejectedRequests, stateTransitions, currentState)

	metrics := &circuitbreaker.PrometheusCircuitBreakerMetricsCollector{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successfulRequests,
		FailedRequests:     failedRequests,
		RejectedRequests:   rejectedRequests,
		StateTransitions:   stateTransitions,
		CurrentState:       *currentState,
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		fmt.Println("Prometheus metrics at :2115/metrics")
		http.ListenAndServe(":2115", nil)
	}()

	cb := circuitbreaker.NewWithOptions(circuitbreaker.Options{}, metrics)

	// Simulate requests
	for i := 0; i < 20; i++ {
		err := cb.Do(func() error {
			if i%4 == 0 {
				return fmt.Errorf("simulated error")
			}
			return nil
		})
		if err != nil {
			fmt.Printf("Request %d failed: %v\n", i, err)
		} else {
			fmt.Printf("Request %d succeeded\n", i)
		}
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println("Circuit Breaker Prometheus example complete. Metrics at :2115/metrics")
	// Keep running to allow Prometheus scraping
	select {
	case <-time.After(5 * time.Second):
	case <-context.Background().Done():
	}
}
