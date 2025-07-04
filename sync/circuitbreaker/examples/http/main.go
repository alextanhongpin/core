// Real-world HTTP client with circuit breaker example
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
)

// APIClient represents a client for calling external APIs
type APIClient struct {
	httpClient *http.Client
	baseURL    string
	cb         *circuitbreaker.Breaker
}

func NewAPIClient(baseURL string, cb *circuitbreaker.Breaker) *APIClient {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	return &APIClient{
		httpClient: client,
		baseURL:    baseURL,
		cb:         cb,
	}
}

func (c *APIClient) GetUser(ctx context.Context, userID string) (string, error) {
	url := fmt.Sprintf("%s/users/%s", c.baseURL, userID)

	var result string
	err := c.cb.Do(func() error {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("server error: %s", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		result = string(body)
		return nil
	})

	return result, err
}

func (c *APIClient) GetCircuitStatus() circuitbreaker.Status {
	return c.cb.Status()
}

// MockAPIServer simulates an unreliable API server
type MockAPIServer struct {
	mu           sync.RWMutex
	failureCount int
	maxFailures  int
	latency      time.Duration
	failureRate  float64
}

func NewMockAPIServer(maxFailures int, latency time.Duration) *MockAPIServer {
	return &MockAPIServer{
		maxFailures: maxFailures,
		latency:     latency,
	}
}

func (s *MockAPIServer) SetFailureRate(rate float64) {
	s.mu.Lock()
	s.failureRate = rate
	s.mu.Unlock()
}

func (s *MockAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	latency := s.latency
	failureRate := s.failureRate
	s.mu.Unlock()

	// Simulate latency
	time.Sleep(latency)

	// Check if we should fail
	if failureRate > 0 && s.failureCount < s.maxFailures {
		s.mu.Lock()
		s.failureCount++
		s.mu.Unlock()

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Server error #%d", s.failureCount)
		return
	}

	// Success response
	userID := r.URL.Path[len("/users/"):]
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"id": "%s", "name": "User %s", "status": "active"}`, userID, userID)
}

func main() {
	fmt.Println("🌐 HTTP Client with Circuit Breaker Example")
	fmt.Println("===========================================")

	// Create mock server that fails first 5 requests
	mockServer := NewMockAPIServer(5, 100*time.Millisecond)
	mockServer.SetFailureRate(1.0) // 100% failure rate initially

	server := httptest.NewServer(mockServer)
	defer server.Close()

	fmt.Printf("🖥️  Mock API server running at: %s\n", server.URL)

	// Create circuit breaker
	cb := circuitbreaker.New()
	cb.BreakDuration = 2 * time.Second
	cb.FailureThreshold = 3
	cb.FailureRatio = 0.7
	cb.SuccessThreshold = 2

	// Add state change monitoring
	cb.OnStateChange = func(old, new circuitbreaker.Status) {
		fmt.Printf("🔄 Circuit breaker: %s -> %s\n", old, new)
	}

	// Create API client
	apiClient := NewAPIClient(server.URL, cb)

	// Phase 1: High failure rate
	fmt.Println("\n📍 Phase 1: API server is failing")
	for i := 1; i <= 8; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

		result, err := apiClient.GetUser(ctx, fmt.Sprintf("user%d", i))

		if errors.Is(err, circuitbreaker.ErrBrokenCircuit) {
			fmt.Printf("Request %d: ⚡ Circuit breaker blocked the request\n", i)
		} else if err != nil {
			fmt.Printf("Request %d: ❌ %v\n", i, err)
		} else {
			fmt.Printf("Request %d: ✅ %s\n", i, result)
		}

		cancel()
		time.Sleep(300 * time.Millisecond)
	}

	// Phase 2: Improve server reliability
	fmt.Println("\n📍 Phase 2: API server is recovering")
	mockServer.SetFailureRate(0.0) // Stop failing

	// Wait for circuit to attempt half-open
	time.Sleep(cb.BreakDuration + 500*time.Millisecond)

	for i := 9; i <= 15; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

		result, err := apiClient.GetUser(ctx, fmt.Sprintf("user%d", i))

		if errors.Is(err, circuitbreaker.ErrBrokenCircuit) {
			fmt.Printf("Request %d: ⚡ Circuit breaker blocked the request\n", i)
		} else if err != nil {
			fmt.Printf("Request %d: ❌ %v\n", i, err)
		} else {
			fmt.Printf("Request %d: ✅ %s\n", i, result)
		}

		cancel()
		time.Sleep(300 * time.Millisecond)
	}

	// Phase 3: Test concurrent requests
	fmt.Println("\n📍 Phase 3: Testing concurrent requests")

	var wg sync.WaitGroup
	results := make(chan string, 10)
	errChan := make(chan error, 10)

	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			result, err := apiClient.GetUser(ctx, fmt.Sprintf("concurrent-user%d", userID))
			if err != nil {
				errChan <- err
			} else {
				results <- result
			}
		}(i)
	}

	wg.Wait()
	close(results)
	close(errChan)

	successCount := 0
	errorCount := 0
	circuitBreakCount := 0

	for result := range results {
		successCount++
		fmt.Printf("Concurrent request: ✅ %s\n", result)
	}

	for err := range errChan {
		errorCount++
		if errors.Is(err, circuitbreaker.ErrBrokenCircuit) {
			circuitBreakCount++
			fmt.Printf("Concurrent request: ⚡ Circuit breaker blocked\n")
		} else {
			fmt.Printf("Concurrent request: ❌ %v\n", err)
		}
	}

	fmt.Printf("\n📊 Concurrent request results:\n")
	fmt.Printf("  ✅ Successful: %d\n", successCount)
	fmt.Printf("  ❌ Failed: %d\n", errorCount)
	fmt.Printf("  ⚡ Circuit breaks: %d\n", circuitBreakCount)

	fmt.Printf("\n🔌 Final circuit breaker status: %s\n", apiClient.GetCircuitStatus())
	fmt.Println("\n✅ HTTP Client Example Complete!")
}
