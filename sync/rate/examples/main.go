package examples

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
)

// ExternalAPIClient demonstrates circuit breaker pattern for external API calls
type ExternalAPIClient struct {
	baseURL string
	client  *http.Client
	breaker *rate.Limiter
}

func NewExternalAPIClient(baseURL string) *ExternalAPIClient {
	// Create circuit breaker that opens after 3 consecutive failures
	breaker := rate.NewLimiter(3)
	breaker.FailureToken = 1.0
	breaker.SuccessToken = 1.0 // Full recovery on success

	return &ExternalAPIClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
		breaker: breaker,
	}
}

func (c *ExternalAPIClient) MakeRequest(ctx context.Context, endpoint string) (*http.Response, error) {
	var resp *http.Response
	var lastErr error

	err := c.breaker.Do(func() error {
		var err error
		err = c.doRequest(ctx, endpoint)
		lastErr = err
		return err
	})

	if err == rate.ErrLimitExceeded {
		return nil, fmt.Errorf("circuit breaker open: %v", lastErr)
	}

	return resp, err
}

func (c *ExternalAPIClient) doRequest(ctx context.Context, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("server error: %d", resp.StatusCode)
	}

	return nil
}

func (c *ExternalAPIClient) GetStats() (success, failure, total int) {
	return c.breaker.Success(), c.breaker.Failure(), c.breaker.Total()
}

// DatabaseService demonstrates rate limiting for database operations
type DatabaseService struct {
	connectionPool *rate.Limiter
	errorTracker   *rate.Errors
}

func NewDatabaseService(maxConcurrentConnections float64) *DatabaseService {
	// Limit concurrent database connections
	pool := rate.NewLimiter(maxConcurrentConnections)
	pool.FailureToken = 2.0 // Database errors are expensive
	pool.SuccessToken = 0.5 // Slow recovery

	return &DatabaseService{
		connectionPool: pool,
		errorTracker:   rate.NewErrors(time.Minute),
	}
}

func (db *DatabaseService) Query(query string) error {
	return db.connectionPool.Do(func() error {
		// Simulate database query
		err := db.simulateQuery(query)

		// Track for monitoring
		if err != nil {
			db.errorTracker.Failure().Inc()
		} else {
			db.errorTracker.Success().Inc()
		}

		return err
	})
}

func (db *DatabaseService) simulateQuery(query string) error {
	// Simulate query time
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

	// Simulate 20% failure rate
	if rand.Float64() < 0.2 {
		return fmt.Errorf("database error: connection timeout")
	}

	return nil
}

func (db *DatabaseService) GetHealth() (successRate, errorRate, errorRatio float64) {
	health := db.errorTracker.Rate()
	return health.Success(), health.Failure(), health.Ratio()
}

func (db *DatabaseService) IsHealthy() bool {
	health := db.errorTracker.Rate()
	// Consider unhealthy if error rate > 15% and we have enough data
	return health.Total() < 5 || health.Ratio() < 0.15
}

// LoadBalancer demonstrates weighted load balancing with health checks
type LoadBalancer struct {
	backends []Backend
}

type Backend struct {
	Name        string
	URL         string
	Weight      float64
	RateTracker *rate.Rate
	HealthCheck *rate.Errors
}

func NewLoadBalancer() *LoadBalancer {
	backends := []Backend{
		{
			Name:        "backend-1",
			URL:         "http://backend1:8080",
			Weight:      1.0,
			RateTracker: rate.NewRate(time.Minute),
			HealthCheck: rate.NewErrors(time.Minute),
		},
		{
			Name:        "backend-2",
			URL:         "http://backend2:8080",
			Weight:      1.0,
			RateTracker: rate.NewRate(time.Minute),
			HealthCheck: rate.NewErrors(time.Minute),
		},
		{
			Name:        "backend-3",
			URL:         "http://backend3:8080",
			Weight:      0.5, // Lower capacity backend
			RateTracker: rate.NewRate(time.Minute),
			HealthCheck: rate.NewErrors(time.Minute),
		},
	}

	return &LoadBalancer{backends: backends}
}

func (lb *LoadBalancer) SelectBackend() *Backend {
	var best *Backend
	var bestScore float64

	for i := range lb.backends {
		backend := &lb.backends[i]

		// Skip unhealthy backends
		health := backend.HealthCheck.Rate()
		if health.Total() > 10 && health.Ratio() > 0.3 {
			continue
		}

		// Calculate score: weight / current_load
		currentLoad := backend.RateTracker.Count()
		score := backend.Weight / (currentLoad + 1)

		if best == nil || score > bestScore {
			best = backend
			bestScore = score
		}
	}

	return best
}

func (lb *LoadBalancer) RouteRequest(ctx context.Context, request string) error {
	backend := lb.SelectBackend()
	if backend == nil {
		return fmt.Errorf("no healthy backends available")
	}

	// Track request to this backend
	backend.RateTracker.Inc()

	// Simulate request
	err := lb.makeRequest(ctx, backend, request)

	// Track health
	if err != nil {
		backend.HealthCheck.Failure().Inc()
	} else {
		backend.HealthCheck.Success().Inc()
	}

	return err
}

func (lb *LoadBalancer) makeRequest(ctx context.Context, backend *Backend, request string) error {
	// Simulate request processing
	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)

	// Simulate backend-specific failure rates
	var failureRate float64
	switch backend.Name {
	case "backend-1":
		failureRate = 0.05 // 5% failure rate
	case "backend-2":
		failureRate = 0.10 // 10% failure rate
	case "backend-3":
		failureRate = 0.20 // 20% failure rate (less reliable)
	}

	if rand.Float64() < failureRate {
		return fmt.Errorf("backend %s failed to process request", backend.Name)
	}

	return nil
}

func (lb *LoadBalancer) GetStats() []BackendStats {
	var stats []BackendStats

	for _, backend := range lb.backends {
		health := backend.HealthCheck.Rate()
		stats = append(stats, BackendStats{
			Name:        backend.Name,
			URL:         backend.URL,
			Weight:      backend.Weight,
			CurrentLoad: backend.RateTracker.Count(),
			SuccessRate: health.Success(),
			FailureRate: health.Failure(),
			ErrorRatio:  health.Ratio(),
			IsHealthy:   health.Total() < 10 || health.Ratio() < 0.3,
		})
	}

	return stats
}

type BackendStats struct {
	Name        string  `json:"name"`
	URL         string  `json:"url"`
	Weight      float64 `json:"weight"`
	CurrentLoad float64 `json:"current_load"`
	SuccessRate float64 `json:"success_rate"`
	FailureRate float64 `json:"failure_rate"`
	ErrorRatio  float64 `json:"error_ratio"`
	IsHealthy   bool    `json:"is_healthy"`
}

// RunExamples demonstrates various usage patterns of the rate limiting library
func RunExamples() {
	// Example 1: External API with Circuit Breaker
	fmt.Println("=== External API Circuit Breaker Demo ===")
	apiClient := NewExternalAPIClient("https://httpbin.org")

	for i := 0; i < 10; i++ {
		_, err := apiClient.MakeRequest(context.Background(), "/status/500") // Always fails
		success, failure, total := apiClient.GetStats()

		fmt.Printf("Request %d: %v | Stats - Success: %d, Failure: %d, Total: %d\n",
			i+1, err, success, failure, total)
		time.Sleep(500 * time.Millisecond)
	}

	// Example 2: Database Service
	fmt.Println("\n=== Database Service Demo ===")
	dbService := NewDatabaseService(5) // Max 5 concurrent connections

	for i := 0; i < 15; i++ {
		err := dbService.Query("SELECT * FROM users")
		successRate, errorRate, errorRatio := dbService.GetHealth()

		fmt.Printf("Query %d: %v | Success/min: %.1f, Error/min: %.1f, Error ratio: %.1f%%, Healthy: %v\n",
			i+1, err, successRate, errorRate, errorRatio*100, dbService.IsHealthy())
		time.Sleep(300 * time.Millisecond)
	}

	// Example 3: Load Balancer
	fmt.Println("\n=== Load Balancer Demo ===")
	lb := NewLoadBalancer()

	for i := 0; i < 20; i++ {
		err := lb.RouteRequest(context.Background(), fmt.Sprintf("request-%d", i+1))
		if err != nil {
			fmt.Printf("Request %d failed: %v\n", i+1, err)
		} else {
			fmt.Printf("Request %d succeeded\n", i+1)
		}

		// Print stats every 5 requests
		if (i+1)%5 == 0 {
			fmt.Println("\nBackend Stats:")
			for _, stats := range lb.GetStats() {
				fmt.Printf("  %s: Load=%.1f, Success=%.1f/min, Error=%.1f%%, Healthy=%v\n",
					stats.Name, stats.CurrentLoad, stats.SuccessRate,
					stats.ErrorRatio*100, stats.IsHealthy)
			}
			fmt.Println()
		}

		time.Sleep(200 * time.Millisecond)
	}
}
