package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
)

// ServiceClient represents a service client that can fail
type ServiceClient struct {
	name        string
	failureRate float64
	latency     time.Duration
	mu          sync.RWMutex
}

func NewServiceClient(name string, failureRate float64, latency time.Duration) *ServiceClient {
	return &ServiceClient{
		name:        name,
		failureRate: failureRate,
		latency:     latency,
	}
}

func (s *ServiceClient) SetFailureRate(rate float64) {
	s.mu.Lock()
	s.failureRate = rate
	s.mu.Unlock()
}

func (s *ServiceClient) SetLatency(latency time.Duration) {
	s.mu.Lock()
	s.latency = latency
	s.mu.Unlock()
}

func (s *ServiceClient) Call(ctx context.Context, data string) (string, error) {
	s.mu.RLock()
	failureRate := s.failureRate
	latency := s.latency
	s.mu.RUnlock()

	// Simulate latency
	time.Sleep(latency)

	// Simulate failures
	if rand.Float64() < failureRate {
		return "", errors.New("service temporarily unavailable")
	}

	return fmt.Sprintf("Processed: %s", data), nil
}

// HTTPService represents an HTTP service with circuit breaker
type HTTPService struct {
	client *http.Client
	cb     *circuitbreaker.Breaker
	server *httptest.Server
}

func NewHTTPService() *HTTPService {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate 50% failure rate initially
		if rand.Float64() < 0.5 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "data": "success"}`))
	}))

	// Create circuit breaker with custom options
	cb := circuitbreaker.NewWithOptions(circuitbreaker.Options{
		FailureThreshold: 3,
		FailureRatio:     0.5,
		BreakDuration:    2 * time.Second,
		SuccessThreshold: 2,
		OnStateChange: func(old, new circuitbreaker.Status) {
			fmt.Printf("🔄 Circuit breaker: %s -> %s\n", old, new)
		},
		OnRequest: func() {
			fmt.Print("📤 Making request... ")
		},
		OnSuccess: func(duration time.Duration) {
			fmt.Printf("✅ Request succeeded in %v\n", duration)
		},
		OnFailure: func(err error, duration time.Duration) {
			fmt.Printf("❌ Request failed in %v: %v\n", duration, err)
		},
		OnReject: func() {
			fmt.Println("⚡ Request rejected - circuit breaker is open")
		},
	})

	return &HTTPService{
		client: &http.Client{Timeout: 5 * time.Second},
		cb:     cb,
		server: server,
	}
}

func (h *HTTPService) Close() {
	h.server.Close()
}

func (h *HTTPService) MakeRequest(ctx context.Context) (string, error) {
	var result string
	err := h.cb.Do(func() error {
		req, err := http.NewRequestWithContext(ctx, "GET", h.server.URL, nil)
		if err != nil {
			return err
		}

		resp, err := h.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		}

		buf := make([]byte, 1024)
		n, err := resp.Body.Read(buf)
		if err != nil && err.Error() != "EOF" {
			return err
		}

		result = string(buf[:n])
		return nil
	})

	return result, err
}

func main() {
	fmt.Println("🚀 Advanced Circuit Breaker Example")
	fmt.Println("===================================")

	// Example 1: Basic service with dynamic failure rate
	fmt.Println("\n📍 Example 1: Service with Dynamic Failure Rate")
	fmt.Println("----------------------------------------------")

	service := NewServiceClient("UserService", 0.8, 100*time.Millisecond)

	// Create circuit breaker with callbacks
	cb := circuitbreaker.NewWithOptions(circuitbreaker.Options{
		FailureThreshold: 3,
		FailureRatio:     0.5,
		BreakDuration:    2 * time.Second,
		SuccessThreshold: 2,
		OnStateChange: func(old, new circuitbreaker.Status) {
			fmt.Printf("🔄 Circuit breaker: %s -> %s\n", old, new)
		},
		OnReject: func() {
			fmt.Println("⚡ Circuit breaker blocked the request")
		},
	})

	// Make some requests that will likely fail
	for i := 1; i <= 10; i++ {
		err := cb.Do(func() error {
			_, err := service.Call(context.Background(), fmt.Sprintf("request-%d", i))
			return err
		})

		if err == circuitbreaker.ErrBrokenCircuit {
			fmt.Printf("Request %d: ⚡ Circuit breaker is open\n", i)
		} else if err != nil {
			fmt.Printf("Request %d: ❌ %v\n", i, err)
		} else {
			fmt.Printf("Request %d: ✅ Success\n", i)
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Improve service reliability
	fmt.Println("\n🌟 Improving service reliability...")
	service.SetFailureRate(0.1)

	// Wait for circuit breaker to transition to half-open
	time.Sleep(2 * time.Second)

	// Make more requests
	for i := 11; i <= 20; i++ {
		err := cb.Do(func() error {
			_, err := service.Call(context.Background(), fmt.Sprintf("request-%d", i))
			return err
		})

		if err == circuitbreaker.ErrBrokenCircuit {
			fmt.Printf("Request %d: ⚡ Circuit breaker is open\n", i)
		} else if err != nil {
			fmt.Printf("Request %d: ❌ %v\n", i, err)
		} else {
			fmt.Printf("Request %d: ✅ Success\n", i)
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Example 2: HTTP service with circuit breaker
	fmt.Println("\n📍 Example 2: HTTP Service with Circuit Breaker")
	fmt.Println("----------------------------------------------")

	httpService := NewHTTPService()
	defer httpService.Close()

	// Make HTTP requests
	for i := 1; i <= 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		result, err := httpService.MakeRequest(ctx)
		cancel()

		if err == circuitbreaker.ErrBrokenCircuit {
			fmt.Printf("HTTP Request %d: ⚡ Circuit breaker is open\n", i)
		} else if err != nil {
			fmt.Printf("HTTP Request %d: ❌ %v\n", i, err)
		} else {
			fmt.Printf("HTTP Request %d: ✅ %s\n", i, result)
		}

		time.Sleep(200 * time.Millisecond)
	}

	// Example 3: Metrics and monitoring
	fmt.Println("\n📍 Example 3: Metrics and Monitoring")
	fmt.Println("-----------------------------------")

	metrics := cb.Metrics()
	fmt.Printf("📊 Circuit Breaker Metrics:\n")
	fmt.Printf("   Total Requests: %d\n", metrics.TotalRequests)
	fmt.Printf("   Successful Requests: %d\n", metrics.SuccessfulRequests)
	fmt.Printf("   Failed Requests: %d\n", metrics.FailedRequests)
	fmt.Printf("   Rejected Requests: %d\n", metrics.RejectedRequests)
	fmt.Printf("   State Transitions: %d\n", metrics.StateTransitions)
	fmt.Printf("   Current State: %s\n", metrics.CurrentState)

	fmt.Println("\n✅ Advanced Circuit Breaker Example Complete!")
}
