package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/alextanhongpin/core/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/slog"
)

// MicroserviceMonitor demonstrates telemetry for monitoring distributed systems
type MicroserviceMonitor struct {
	telemetry     *telemetry.MultiHandler
	ctx           context.Context
	prometheusReg *prometheus.Registry
	mu            sync.RWMutex
	services      map[string]*ServiceHealth

	// System metrics
	requestsTotal       *event.Counter
	requestDuration     *event.DurationDistribution
	errorRate           *event.Counter
	serviceHealth       *event.FloatGauge
	resourceUsage       *event.FloatGauge
	throughput          *event.Counter
	circuitBreakerState *event.Counter
}

type ServiceHealth struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	LastCheck    time.Time         `json:"last_check"`
	ResponseTime time.Duration     `json:"response_time"`
	ErrorCount   int               `json:"error_count"`
	Uptime       float64           `json:"uptime_percentage"`
	Metadata     map[string]string `json:"metadata"`
}

type CircuitBreaker struct {
	name        string
	failures    int
	lastFailure time.Time
	state       string // "closed", "open", "half-open"
	threshold   int
	timeout     time.Duration
	mu          sync.Mutex
}

func NewMicroserviceMonitor() (*MicroserviceMonitor, error) {
	// Setup enhanced logging for microservices
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false, // Reduce noise for high-volume microservice logs
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String("@timestamp", a.Value.Time().Format(time.RFC3339Nano))
			}
			if a.Key == slog.LevelKey {
				return slog.String("level", a.Value.String())
			}
			return a
		},
	}))

	slogHandler, err := telemetry.NewSlogHandler(logger, telemetry.WithSlogErrorHandler(func(err error) {
		logger.Error("microservice telemetry error", "error", err, "component", "telemetry")
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create slog handler: %w", err)
	}

	prometheusReg := prometheus.NewRegistry()
	prometheusHandler, err := telemetry.NewPrometheusHandler(prometheusReg, telemetry.WithPrometheusErrorHandler(func(err error) {
		logger.Error("prometheus telemetry error", "error", err, "component", "prometheus")
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus handler: %w", err)
	}

	multiHandler := &telemetry.MultiHandler{
		Log:    slogHandler,
		Metric: prometheusHandler,
	}

	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(multiHandler, eventtest.ExporterOptions()))

	monitor := &MicroserviceMonitor{
		telemetry:     multiHandler,
		ctx:           ctx,
		prometheusReg: prometheusReg,
		services:      make(map[string]*ServiceHealth),
	}

	monitor.initializeMetrics()
	return monitor, nil
}

func (mm *MicroserviceMonitor) initializeMetrics() {
	// Microservice-specific metrics
	mm.requestsTotal = event.NewCounter("microservice_requests_total", &event.MetricOptions{
		Description: "Total HTTP requests across all services",
		Namespace:   "microservice",
	})

	mm.requestDuration = event.NewDuration("microservice_request_duration", &event.MetricOptions{
		Description: "Request duration distribution",
		Namespace:   "microservice",
	})

	mm.errorRate = event.NewCounter("microservice_errors_total", &event.MetricOptions{
		Description: "Total errors by service and type",
		Namespace:   "microservice",
	})

	mm.serviceHealth = event.NewFloatGauge("microservice_health_score", &event.MetricOptions{
		Description: "Service health score (0-1)",
		Namespace:   "microservice",
	})

	mm.resourceUsage = event.NewFloatGauge("microservice_resource_usage", &event.MetricOptions{
		Description: "Resource usage metrics",
		Namespace:   "microservice",
	})

	mm.throughput = event.NewCounter("microservice_throughput_ops", &event.MetricOptions{
		Description: "Operations throughput per service",
		Namespace:   "microservice",
	})

	mm.circuitBreakerState = event.NewCounter("microservice_circuit_breaker_events", &event.MetricOptions{
		Description: "Circuit breaker state changes",
		Namespace:   "microservice",
	})
}

// SimulateDistributedRequest demonstrates telemetry for distributed system calls
func (mm *MicroserviceMonitor) SimulateDistributedRequest(userID, requestID string) error {
	start := time.Now()

	// Create distributed tracing context
	ctx := mm.ctx
	ctx = event.WithExporter(ctx, event.NewExporter(mm.telemetry, eventtest.ExporterOptions()))

	event.Log(ctx, "distributed request initiated",
		event.String("request_id", requestID),
		event.String("user_id", userID),
		event.String("service", "api-gateway"),
		event.String("operation", "process_user_request"),
	)

	// Step 1: Authentication service call
	authStart := time.Now()
	if err := mm.callAuthService(ctx, userID, requestID); err != nil {
		mm.recordServiceError(ctx, "auth-service", "authentication_failed", err)
		return fmt.Errorf("authentication failed: %w", err)
	}
	authDuration := time.Since(authStart)
	mm.recordServiceCall(ctx, "auth-service", authDuration, "success")

	// Step 2: User service call
	userStart := time.Now()
	userData, err := mm.callUserService(ctx, userID, requestID)
	if err != nil {
		mm.recordServiceError(ctx, "user-service", "user_lookup_failed", err)
		return fmt.Errorf("user lookup failed: %w", err)
	}
	userDuration := time.Since(userStart)
	mm.recordServiceCall(ctx, "user-service", userDuration, "success")

	// Step 3: Permission service call
	permStart := time.Now()
	if err := mm.callPermissionService(ctx, userID, userData["role"].(string), requestID); err != nil {
		mm.recordServiceError(ctx, "permission-service", "authorization_failed", err)
		return fmt.Errorf("authorization failed: %w", err)
	}
	permDuration := time.Since(permStart)
	mm.recordServiceCall(ctx, "permission-service", permDuration, "success")

	// Step 4: Business logic service call
	bizStart := time.Now()
	result, err := mm.callBusinessService(ctx, userID, requestID)
	if err != nil {
		mm.recordServiceError(ctx, "business-service", "business_logic_failed", err)
		return fmt.Errorf("business logic failed: %w", err)
	}
	bizDuration := time.Since(bizStart)
	mm.recordServiceCall(ctx, "business-service", bizDuration, "success")

	// Step 5: Audit service call (fire-and-forget)
	go mm.callAuditService(ctx, userID, requestID, result)

	// Record overall request metrics
	totalDuration := time.Since(start)
	mm.requestsTotal.Record(ctx, 1,
		event.String("service", "api-gateway"),
		event.String("status", "success"),
	)

	mm.requestDuration.Record(ctx, totalDuration,
		event.String("service", "api-gateway"),
		event.String("user_type", getUserTypeFromID(userID)),
	)

	event.Log(ctx, "distributed request completed",
		event.String("request_id", requestID),
		event.String("user_id", userID),
		event.Int64("total_duration_ms", totalDuration.Milliseconds()),
		event.Int64("auth_duration_ms", authDuration.Milliseconds()),
		event.Int64("user_duration_ms", userDuration.Milliseconds()),
		event.Int64("perm_duration_ms", permDuration.Milliseconds()),
		event.Int64("business_duration_ms", bizDuration.Milliseconds()),
		event.String("result", "success"),
	)

	return nil
}

func (mm *MicroserviceMonitor) callAuthService(ctx context.Context, userID, requestID string) error {
	serviceName := "auth-service"

	event.Log(ctx, "calling authentication service",
		event.String("service", serviceName),
		event.String("user_id", userID),
		event.String("request_id", requestID),
		event.String("operation", "validate_token"),
	)

	// Simulate network call
	time.Sleep(time.Duration(20+rand.Intn(30)) * time.Millisecond)

	// Simulate authentication failures (2% rate)
	if rand.Float64() < 0.02 {
		return fmt.Errorf("invalid authentication token")
	}

	event.Log(ctx, "authentication successful",
		event.String("service", serviceName),
		event.String("user_id", userID),
		event.String("request_id", requestID),
	)

	return nil
}

func (mm *MicroserviceMonitor) callUserService(ctx context.Context, userID, requestID string) (map[string]interface{}, error) {
	serviceName := "user-service"

	event.Log(ctx, "calling user service",
		event.String("service", serviceName),
		event.String("user_id", userID),
		event.String("request_id", requestID),
		event.String("operation", "get_user_profile"),
	)

	// Simulate database lookup
	time.Sleep(time.Duration(30+rand.Intn(50)) * time.Millisecond)

	// Simulate user not found (1% rate)
	if rand.Float64() < 0.01 {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	// Mock user data
	userData := map[string]interface{}{
		"id":    userID,
		"name":  fmt.Sprintf("User_%s", userID),
		"email": fmt.Sprintf("%s@example.com", userID),
		"role":  "user",
	}

	if rand.Float64() < 0.1 { // 10% are admins
		userData["role"] = "admin"
	}

	event.Log(ctx, "user data retrieved",
		event.String("service", serviceName),
		event.String("user_id", userID),
		event.String("request_id", requestID),
		event.String("user_role", userData["role"].(string)),
	)

	return userData, nil
}

func (mm *MicroserviceMonitor) callPermissionService(ctx context.Context, userID, role, requestID string) error {
	serviceName := "permission-service"

	event.Log(ctx, "calling permission service",
		event.String("service", serviceName),
		event.String("user_id", userID),
		event.String("user_role", role),
		event.String("request_id", requestID),
		event.String("operation", "check_permissions"),
	)

	// Simulate permission check
	time.Sleep(time.Duration(15+rand.Intn(25)) * time.Millisecond)

	// Simulate permission denied (1% rate for regular users, 0.1% for admins)
	denyRate := 0.01
	if role == "admin" {
		denyRate = 0.001
	}

	if rand.Float64() < denyRate {
		return fmt.Errorf("access denied for user %s with role %s", userID, role)
	}

	event.Log(ctx, "permission check passed",
		event.String("service", serviceName),
		event.String("user_id", userID),
		event.String("user_role", role),
		event.String("request_id", requestID),
	)

	return nil
}

func (mm *MicroserviceMonitor) callBusinessService(ctx context.Context, userID, requestID string) (map[string]interface{}, error) {
	serviceName := "business-service"

	event.Log(ctx, "calling business service",
		event.String("service", serviceName),
		event.String("user_id", userID),
		event.String("request_id", requestID),
		event.String("operation", "process_business_logic"),
	)

	// Simulate complex business logic
	time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)

	// Simulate business logic failures (3% rate)
	if rand.Float64() < 0.03 {
		return nil, fmt.Errorf("business rule validation failed")
	}

	result := map[string]interface{}{
		"processed":   true,
		"timestamp":   time.Now().Unix(),
		"result_code": 200,
		"data":        fmt.Sprintf("processed_data_for_%s", userID),
	}

	event.Log(ctx, "business logic completed",
		event.String("service", serviceName),
		event.String("user_id", userID),
		event.String("request_id", requestID),
		event.String("result_code", "200"),
	)

	return result, nil
}

func (mm *MicroserviceMonitor) callAuditService(ctx context.Context, userID, requestID string, result map[string]interface{}) {
	serviceName := "audit-service"

	event.Log(ctx, "calling audit service",
		event.String("service", serviceName),
		event.String("user_id", userID),
		event.String("request_id", requestID),
		event.String("operation", "log_user_action"),
	)

	// Simulate audit logging
	time.Sleep(time.Duration(10+rand.Intn(20)) * time.Millisecond)

	event.Log(ctx, "audit record created",
		event.String("service", serviceName),
		event.String("user_id", userID),
		event.String("request_id", requestID),
		event.String("audit_result", "logged"),
	)
}

func (mm *MicroserviceMonitor) recordServiceCall(ctx context.Context, serviceName string, duration time.Duration, status string) {
	mm.requestsTotal.Record(ctx, 1,
		event.String("service", serviceName),
		event.String("status", status),
	)

	mm.requestDuration.Record(ctx, duration,
		event.String("service", serviceName),
		event.String("status", status),
	)

	mm.throughput.Record(ctx, 1,
		event.String("service", serviceName),
	)

	// Update service health
	mm.updateServiceHealth(serviceName, duration, status == "success")
}

func (mm *MicroserviceMonitor) recordServiceError(ctx context.Context, serviceName, errorType string, err error) {
	mm.errorRate.Record(ctx, 1,
		event.String("service", serviceName),
		event.String("error_type", errorType),
	)

	event.Log(ctx, "service error occurred",
		event.String("service", serviceName),
		event.String("error_type", errorType),
		event.String("error_message", err.Error()),
		event.String("severity", "error"),
	)

	// Update service health for errors
	mm.updateServiceHealth(serviceName, 0, false)
}

func (mm *MicroserviceMonitor) updateServiceHealth(serviceName string, responseTime time.Duration, success bool) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	health, exists := mm.services[serviceName]
	if !exists {
		health = &ServiceHealth{
			Name:     serviceName,
			Status:   "healthy",
			Uptime:   100.0,
			Metadata: make(map[string]string),
		}
		mm.services[serviceName] = health
	}

	health.LastCheck = time.Now()
	health.ResponseTime = responseTime

	if success {
		// Improve health score
		health.Uptime = minFloat(100.0, health.Uptime+0.1)
		if health.Uptime > 95.0 {
			health.Status = "healthy"
		} else if health.Uptime > 80.0 {
			health.Status = "degraded"
		}
	} else {
		// Degrade health score
		health.ErrorCount++
		health.Uptime = maxFloat(0.0, health.Uptime-2.0)
		if health.Uptime < 50.0 {
			health.Status = "unhealthy"
		} else if health.Uptime < 80.0 {
			health.Status = "degraded"
		}
	}

	// Record health metric
	healthScore := health.Uptime / 100.0
	mm.serviceHealth.Record(mm.ctx, healthScore,
		event.String("service", serviceName),
		event.String("status", health.Status),
	)
}

// CircuitBreaker implementation for resilient microservice calls
func NewCircuitBreaker(name string, threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:      name,
		threshold: threshold,
		timeout:   timeout,
		state:     "closed",
	}
}

func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error, monitor *MicroserviceMonitor) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if circuit should be reset
	if cb.state == "open" && time.Since(cb.lastFailure) > cb.timeout {
		cb.state = "half-open"
		monitor.circuitBreakerState.Record(ctx, 1,
			event.String("circuit_breaker", cb.name),
			event.String("state", "half-open"),
		)
		event.Log(ctx, "circuit breaker transitioning to half-open",
			event.String("circuit_breaker", cb.name),
			event.String("previous_state", "open"),
		)
	}

	// Reject if circuit is open
	if cb.state == "open" {
		monitor.circuitBreakerState.Record(ctx, 1,
			event.String("circuit_breaker", cb.name),
			event.String("state", "rejected"),
		)
		return fmt.Errorf("circuit breaker %s is open", cb.name)
	}

	// Execute function
	err := fn()

	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()

		// Open circuit if threshold reached
		if cb.failures >= cb.threshold {
			cb.state = "open"
			monitor.circuitBreakerState.Record(ctx, 1,
				event.String("circuit_breaker", cb.name),
				event.String("state", "open"),
			)
			event.Log(ctx, "circuit breaker opened due to failures",
				event.String("circuit_breaker", cb.name),
				event.Int64("failure_count", int64(cb.failures)),
				event.Int64("threshold", int64(cb.threshold)),
			)
		}
		return err
	}

	// Success - reset failures and close circuit
	if cb.state == "half-open" {
		cb.state = "closed"
		monitor.circuitBreakerState.Record(ctx, 1,
			event.String("circuit_breaker", cb.name),
			event.String("state", "closed"),
		)
		event.Log(ctx, "circuit breaker closed after successful call",
			event.String("circuit_breaker", cb.name),
		)
	}
	cb.failures = 0

	return nil
}

// Monitoring and health check endpoints
func (mm *MicroserviceMonitor) setupMonitoringEndpoints() *http.ServeMux {
	mux := http.NewServeMux()

	// Service health endpoint
	mux.HandleFunc("/health", mm.handleHealthCheck)
	mux.HandleFunc("/health/services", mm.handleServiceHealth)

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.HandlerFor(mm.prometheusReg, promhttp.HandlerOpts{}))

	// Simulate distributed request endpoint
	mux.HandleFunc("/api/request", mm.handleSimulateRequest)

	// Load testing endpoint
	mux.HandleFunc("/api/load-test", mm.handleLoadTest)

	return mux
}

func (mm *MicroserviceMonitor) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = event.WithExporter(ctx, event.NewExporter(mm.telemetry, eventtest.ExporterOptions()))

	event.Log(ctx, "health check requested")

	mm.mu.RLock()
	allHealthy := true
	for _, service := range mm.services {
		if service.Status != "healthy" {
			allHealthy = false
			break
		}
	}
	mm.mu.RUnlock()

	status := "healthy"
	httpStatus := http.StatusOK
	if !allHealthy {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	response := map[string]interface{}{
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "microservice-monitor",
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(response)
}

func (mm *MicroserviceMonitor) handleServiceHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = event.WithExporter(ctx, event.NewExporter(mm.telemetry, eventtest.ExporterOptions()))

	event.Log(ctx, "service health details requested")

	mm.mu.RLock()
	services := make(map[string]*ServiceHealth)
	for name, health := range mm.services {
		services[name] = health
	}
	mm.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"services":  services,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (mm *MicroserviceMonitor) handleSimulateRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = event.WithExporter(ctx, event.NewExporter(mm.telemetry, eventtest.ExporterOptions()))

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = fmt.Sprintf("user_%d", rand.Intn(1000))
	}

	requestID := fmt.Sprintf("req_%d_%d", time.Now().Unix(), rand.Intn(10000))

	event.Log(ctx, "simulating distributed request",
		event.String("user_id", userID),
		event.String("request_id", requestID),
	)

	if err := mm.SimulateDistributedRequest(userID, requestID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      err.Error(),
			"request_id": requestID,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"request_id": requestID,
		"user_id":    userID,
	})
}

func (mm *MicroserviceMonitor) handleLoadTest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = event.WithExporter(ctx, event.NewExporter(mm.telemetry, eventtest.ExporterOptions()))

	event.Log(ctx, "load test initiated")

	// Simulate concurrent requests
	concurrency := 10
	requests := 50

	var wg sync.WaitGroup
	results := make(chan error, requests)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < requests/concurrency; j++ {
				userID := fmt.Sprintf("loadtest_user_%d_%d", workerID, j)
				requestID := fmt.Sprintf("loadtest_req_%d_%d_%d", time.Now().Unix(), workerID, j)

				err := mm.SimulateDistributedRequest(userID, requestID)
				results <- err

				// Small delay between requests
				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Collect results
	successCount := 0
	errorCount := 0
	for err := range results {
		if err == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	event.Log(ctx, "load test completed",
		event.Int64("total_requests", int64(requests)),
		event.Int64("successful_requests", int64(successCount)),
		event.Int64("failed_requests", int64(errorCount)),
		event.Float64("success_rate", float64(successCount)/float64(requests)*100),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_requests":      requests,
		"successful_requests": successCount,
		"failed_requests":     errorCount,
		"success_rate":        float64(successCount) / float64(requests) * 100,
	})
}

// Helper functions
func getUserTypeFromID(userID string) string {
	if len(userID) > 10 {
		return "premium"
	}
	return "standard"
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// Demo function
func runMicroserviceMonitoringDemo() {
	fmt.Println("🔍 Microservice Monitoring with Advanced Telemetry")
	fmt.Println("=================================================")

	monitor, err := NewMicroserviceMonitor()
	if err != nil {
		fmt.Printf("❌ Failed to create monitor: %v\n", err)
		return
	}

	// Demo 1: Simulate some distributed requests
	fmt.Println("\n🌐 Simulating distributed service calls...")

	testUsers := []string{
		"alice_premium_user",
		"bob_standard_user",
		"charlie_admin_user",
		"diana_enterprise_user",
	}

	for i, userID := range testUsers {
		requestID := fmt.Sprintf("demo_req_%03d", i+1)
		fmt.Printf("Processing request %s for user %s...\n", requestID, userID)

		if err := monitor.SimulateDistributedRequest(userID, requestID); err != nil {
			fmt.Printf("  ❌ Request failed: %v\n", err)
		} else {
			fmt.Printf("  ✅ Request completed successfully\n")
		}

		time.Sleep(200 * time.Millisecond)
	}

	// Demo 2: Circuit breaker example
	fmt.Println("\n⚡ Testing circuit breaker...")

	circuitBreaker := NewCircuitBreaker("demo-service", 3, 5*time.Second)

	// Simulate failing service
	for i := 0; i < 6; i++ {
		err := circuitBreaker.Call(monitor.ctx, func() error {
			if i < 4 { // First 4 calls fail
				return fmt.Errorf("service unavailable")
			}
			return nil // Last 2 succeed
		}, monitor)

		if err != nil {
			fmt.Printf("  ❌ Call %d failed: %v\n", i+1, err)
		} else {
			fmt.Printf("  ✅ Call %d succeeded\n", i+1)
		}
	}

	// Demo 3: Start monitoring server
	fmt.Println("\n🚀 Starting microservice monitoring server...")
	mux := monitor.setupMonitoringEndpoints()

	fmt.Println("Monitoring endpoints available:")
	fmt.Println("  🏥 GET  /health             - Overall health status")
	fmt.Println("  📊 GET  /health/services    - Detailed service health")
	fmt.Println("  📈 GET  /metrics            - Prometheus metrics")
	fmt.Println("  🎯 GET  /api/request        - Simulate distributed request")
	fmt.Println("  🚀 GET  /api/load-test      - Run load test")

	fmt.Println("\n🌐 Server starting on http://localhost:8082")
	fmt.Println("💡 Try these commands:")
	fmt.Println(`  curl "http://localhost:8082/api/request?user_id=test_user"`)
	fmt.Println(`  curl http://localhost:8082/health/services`)
	fmt.Println(`  curl http://localhost:8082/api/load-test`)
	fmt.Println(`  curl http://localhost:8082/metrics`)

	if err := http.ListenAndServe(":8082", mux); err != nil {
		fmt.Printf("❌ Monitoring server failed: %v\n", err)
	}
}

// Uncomment to run as standalone application
func main() {
	runMicroserviceMonitoringDemo()
}
