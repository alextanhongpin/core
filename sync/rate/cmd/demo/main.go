package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
)

func main() {
	fmt.Println("Rate Limiting Library Demo")
	fmt.Println("==========================")

	// Demo 1: Simple Rate Counter
	demoRateCounter()

	// Demo 2: Circuit Breaker
	demoCircuitBreaker()

	// Demo 3: Error Rate Monitoring
	demoErrorRateMonitoring()

	// Demo 4: Combined Usage
	demoCombinedUsage()
}

func demoRateCounter() {
	fmt.Println("\n1. Rate Counter Demo")
	fmt.Println("-------------------")

	// Create a rate counter that tracks requests per second
	counter := rate.NewRate(time.Second)

	fmt.Println("Simulating requests with varying intervals...")

	for i := 0; i < 10; i++ {
		rate := counter.Inc()
		fmt.Printf("Request %2d: %.2f req/s\n", i+1, rate)

		// Vary the sleep time to show rate changes
		if i < 5 {
			time.Sleep(100 * time.Millisecond) // Fast requests
		} else {
			time.Sleep(300 * time.Millisecond) // Slower requests
		}
	}

	// Show rate scaling
	fmt.Printf("Current rate per minute: %.2f req/min\n", counter.Per(time.Minute))
}

func demoCircuitBreaker() {
	fmt.Println("\n2. Circuit Breaker Demo")
	fmt.Println("----------------------")

	// Create a circuit breaker that opens after 3 failures
	limiter := rate.NewLimiter(3)
	limiter.FailureToken = 1.0
	limiter.SuccessToken = 0.5

	fmt.Println("Simulating API calls with failures...")

	for i := 0; i < 15; i++ {
		if !limiter.Allow() {
			fmt.Printf("Call %2d: ❌ BLOCKED (Circuit breaker open)\n", i+1)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Simulate API call with 70% failure rate initially, then improve
		success := rand.Float64() > 0.7
		if i > 10 {
			success = rand.Float64() > 0.2 // Improve success rate later
		}

		if success {
			limiter.Ok()
			fmt.Printf("Call %2d: ✅ SUCCESS\n", i+1)
		} else {
			limiter.Err()
			fmt.Printf("Call %2d: ❌ FAILED\n", i+1)
		}

		fmt.Printf("         Stats: %d success, %d failures, %d total\n",
			limiter.Success(), limiter.Failure(), limiter.Total())

		time.Sleep(200 * time.Millisecond)
	}
}

func demoErrorRateMonitoring() {
	fmt.Println("\n3. Error Rate Monitoring Demo")
	fmt.Println("-----------------------------")

	// Create error tracker with 30-second window
	tracker := rate.NewErrors(30 * time.Second)

	fmt.Println("Simulating database operations...")

	for i := 0; i < 20; i++ {
		// Simulate operations with changing error rates
		var errorRate float64
		switch {
		case i < 5:
			errorRate = 0.1 // 10% error rate initially
		case i < 10:
			errorRate = 0.4 // Spike to 40% error rate
		case i < 15:
			errorRate = 0.2 // Improve to 20%
		default:
			errorRate = 0.05 // Recover to 5%
		}

		if rand.Float64() < errorRate {
			tracker.Failure().Inc()
			fmt.Printf("Op %2d: ❌ FAILED", i+1)
		} else {
			tracker.Success().Inc()
			fmt.Printf("Op %2d: ✅ SUCCESS", i+1)
		}

		// Show current error rate
		errorStats := tracker.Rate()
		if errorStats.Total() > 0 {
			fmt.Printf(" | Error rate: %.1f%% (%.1f success/min, %.1f errors/min)\n",
				errorStats.Ratio()*100, errorStats.Success(), errorStats.Failure())
		} else {
			fmt.Println()
		}

		time.Sleep(300 * time.Millisecond)
	}
}

func demoCombinedUsage() {
	fmt.Println("\n4. Combined Usage Demo - Smart Service")
	fmt.Println("-------------------------------------")

	service := NewSmartService()

	fmt.Println("Simulating service calls with adaptive behavior...")

	for i := 0; i < 25; i++ {
		result := service.ProcessRequest(context.Background(), fmt.Sprintf("request-%d", i+1))

		success, failure, errorRate := service.GetStats()
		healthy := service.IsHealthy()

		statusIcon := "✅"
		if result.Error != nil {
			statusIcon = "❌"
		}
		if result.Blocked {
			statusIcon = "🚫"
		}

		fmt.Printf("Request %2d: %s %-15s | Success: %2d, Failures: %2d, Error Rate: %5.1f%%, Healthy: %v\n",
			i+1, statusIcon, result.Status, success, failure, errorRate*100, healthy)

		time.Sleep(300 * time.Millisecond)
	}
}

// SmartService combines multiple rate limiting strategies
type SmartService struct {
	circuitBreaker *rate.Limiter
	errorTracker   *rate.Errors
	requestRate    *rate.Rate
}

type RequestResult struct {
	Status  string
	Error   error
	Blocked bool
}

func NewSmartService() *SmartService {
	// Circuit breaker opens after 5 failures
	breaker := rate.NewLimiter(5)
	breaker.FailureToken = 1.0
	breaker.SuccessToken = 0.8

	return &SmartService{
		circuitBreaker: breaker,
		errorTracker:   rate.NewErrors(time.Minute),
		requestRate:    rate.NewRate(time.Second),
	}
}

func (s *SmartService) ProcessRequest(ctx context.Context, requestID string) RequestResult {
	// Track request rate
	s.requestRate.Inc()

	// Check circuit breaker
	if !s.circuitBreaker.Allow() {
		return RequestResult{
			Status:  "BLOCKED",
			Error:   fmt.Errorf("circuit breaker open"),
			Blocked: true,
		}
	}

	// Simulate processing with adaptive failure rate
	// Failure rate increases if we're processing too many requests
	baseFailureRate := 0.1
	currentRate := s.requestRate.Count()
	if currentRate > 5 {
		baseFailureRate += (currentRate - 5) * 0.1 // Increase failure rate under load
	}

	// Simulate processing time
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

	if rand.Float64() < baseFailureRate {
		// Request failed
		err := fmt.Errorf("processing failed")
		s.circuitBreaker.Err()
		s.errorTracker.Failure().Inc()

		return RequestResult{
			Status: "FAILED",
			Error:  err,
		}
	}

	// Request succeeded
	s.circuitBreaker.Ok()
	s.errorTracker.Success().Inc()

	return RequestResult{
		Status: "SUCCESS",
	}
}

func (s *SmartService) GetStats() (success, failure int, errorRate float64) {
	success = s.circuitBreaker.Success()
	failure = s.circuitBreaker.Failure()

	errorStats := s.errorTracker.Rate()
	if errorStats.Total() > 0 {
		errorRate = errorStats.Ratio()
	}

	return
}

func (s *SmartService) IsHealthy() bool {
	errorStats := s.errorTracker.Rate()

	// Consider unhealthy if error rate > 20% with sufficient data
	if errorStats.Total() > 5 && errorStats.Ratio() > 0.2 {
		return false
	}

	// Also consider unhealthy if circuit breaker is frequently triggered
	if s.circuitBreaker.Total() > 10 {
		recentFailureRate := float64(s.circuitBreaker.Failure()) / float64(s.circuitBreaker.Total())
		if recentFailureRate > 0.3 {
			return false
		}
	}

	return true
}
