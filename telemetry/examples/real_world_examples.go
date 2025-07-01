// Package main demonstrates real-world usage of the telemetry package
// This file contains multiple examples that can be run individually
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alextanhongpin/core/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/slog"
)

// Example 1: Basic Web Application with Telemetry
func runBasicWebExample() {
	fmt.Println("=== Basic Web Application Example ===")

	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}))

	// Create handlers
	slogHandler, err := telemetry.NewSlogHandler(logger)
	if err != nil {
		log.Fatalf("failed to create slog handler: %v", err)
	}

	prometheusReg := prometheus.NewRegistry()
	prometheusHandler, err := telemetry.NewPrometheusHandler(prometheusReg)
	if err != nil {
		log.Fatalf("failed to create prometheus handler: %v", err)
	}

	// Create unified telemetry handler
	multiHandler := &telemetry.MultiHandler{
		Log:    slogHandler,
		Metric: prometheusHandler,
	}

	// Setup telemetry context
	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(multiHandler, eventtest.ExporterOptions()))

	// Create application metrics
	requestCounter := event.NewCounter("http_requests_total", &event.MetricOptions{
		Description: "Total number of HTTP requests",
	})

	// Start processing some requests
	fmt.Println("Processing sample HTTP requests...")

	for i := 0; i < 5; i++ {
		processHTTPRequest(ctx, requestCounter, fmt.Sprintf("/api/users/%d", i+1), "GET")
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println("✅ Basic web example completed")
}

func processHTTPRequest(ctx context.Context, counter *event.Counter, path, method string) {
	start := time.Now()

	// Log incoming request
	event.Log(ctx, "processing request",
		event.String("method", method),
		event.String("path", path),
		event.String("user_agent", "telemetry-demo/1.0"),
	)

	// Simulate request processing
	processingTime := time.Duration(50+rand.Intn(200)) * time.Millisecond
	time.Sleep(processingTime)

	// Determine status (simulate occasional errors)
	status := "200"
	if rand.Float64() < 0.1 {
		status = "500"
	}

	// Record metrics
	counter.Record(ctx, 1,
		event.String("method", method),
		event.String("path", path),
		event.String("status", status),
	)

	// Log completion
	event.Log(ctx, "request completed",
		event.String("method", method),
		event.String("path", path),
		event.String("status", status),
		event.Int64("duration_ms", time.Since(start).Milliseconds()),
	)
}

// Example 2: Business Logic with Comprehensive Telemetry
func runBusinessLogicExample() {
	fmt.Println("\n=== Business Logic Example ===")

	// Setup telemetry
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}))

	slogHandler, _ := telemetry.NewSlogHandler(logger)
	prometheusReg := prometheus.NewRegistry()
	prometheusHandler, _ := telemetry.NewPrometheusHandler(prometheusReg)

	multiHandler := &telemetry.MultiHandler{
		Log:    slogHandler,
		Metric: prometheusHandler,
	}

	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(multiHandler, eventtest.ExporterOptions()))

	// Process some business operations
	fmt.Println("Processing business operations...")

	operations := []struct {
		userID    string
		operation string
		amount    float64
	}{
		{"user_001", "purchase", 99.99},
		{"user_002", "refund", 49.99},
		{"user_001", "purchase", 199.99},
		{"user_003", "purchase", 29.99},
	}

	for _, op := range operations {
		processBusinessOperation(ctx, op.userID, op.operation, op.amount)
		time.Sleep(300 * time.Millisecond)
	}

	fmt.Println("✅ Business logic example completed")
}

func processBusinessOperation(ctx context.Context, userID, operation string, amount float64) {
	start := time.Now()

	// Log operation start
	event.Log(ctx, "business operation started",
		event.String("user_id", userID),
		event.String("operation", operation),
		event.Float64("amount", amount),
	)

	// Create operation-specific metrics
	operationCounter := event.NewCounter("business_operations_total", &event.MetricOptions{
		Description: "Total number of business operations",
	})

	processingDuration := event.NewDuration("operation_processing_time", &event.MetricOptions{
		Description: "Time taken to process business operations",
	})

	// Simulate validation
	if err := validateOperation(ctx, userID, operation, amount); err != nil {
		event.Log(ctx, "operation validation failed",
			event.String("user_id", userID),
			event.String("operation", operation),
			event.String("error", err.Error()),
		)

		operationCounter.Record(ctx, 1,
			event.String("operation", operation),
			event.String("status", "validation_error"),
		)
		return
	}

	// Simulate processing
	time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)

	// Simulate occasional failures
	if rand.Float64() < 0.15 {
		event.Log(ctx, "operation processing failed",
			event.String("user_id", userID),
			event.String("operation", operation),
			event.String("error", "service_unavailable"),
		)

		operationCounter.Record(ctx, 1,
			event.String("operation", operation),
			event.String("status", "error"),
		)
		return
	}

	// Record successful operation
	duration := time.Since(start)
	operationCounter.Record(ctx, 1,
		event.String("operation", operation),
		event.String("status", "success"),
		event.String("user_type", getUserType(userID)),
	)

	processingDuration.Record(ctx, duration,
		event.String("operation", operation),
		event.String("user_type", getUserType(userID)),
	)

	event.Log(ctx, "business operation completed",
		event.String("user_id", userID),
		event.String("operation", operation),
		event.Float64("amount", amount),
		event.String("status", "success"),
		event.Int64("duration_ms", duration.Milliseconds()),
	)
}

func validateOperation(ctx context.Context, userID, operation string, amount float64) error {
	event.Log(ctx, "validating operation",
		event.String("user_id", userID),
		event.String("operation", operation),
	)

	if userID == "" {
		return errors.New("user ID is required")
	}

	if operation == "" {
		return errors.New("operation type is required")
	}

	if amount < 0 {
		return errors.New("amount cannot be negative")
	}

	if operation == "refund" && amount > 1000 {
		return errors.New("refund amount exceeds limit")
	}

	return nil
}

func getUserType(userID string) string {
	if strings.Contains(userID, "premium") {
		return "premium"
	}
	return "standard"
}

// Example 3: Error Handling and Recovery
func runErrorHandlingExample() {
	fmt.Println("\n=== Error Handling Example ===")

	// Setup telemetry with custom error handlers
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	var slogErrors []error
	slogHandler, _ := telemetry.NewSlogHandler(logger, telemetry.WithSlogErrorHandler(func(err error) {
		slogErrors = append(slogErrors, err)
		logger.Error("slog handler error", "error", err)
	}))

	prometheusReg := prometheus.NewRegistry()
	var prometheusErrors []error
	prometheusHandler, _ := telemetry.NewPrometheusHandler(prometheusReg, telemetry.WithPrometheusErrorHandler(func(err error) {
		prometheusErrors = append(prometheusErrors, err)
		logger.Error("prometheus handler error", "error", err)
	}))

	multiHandler := &telemetry.MultiHandler{
		Log:    slogHandler,
		Metric: prometheusHandler,
	}

	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(multiHandler, eventtest.ExporterOptions()))

	fmt.Println("Testing error handling scenarios...")

	// Test 1: Process nil event (should be handled gracefully)
	multiHandler.Event(ctx, nil)

	// Test 2: Normal operation
	errorCounter := event.NewCounter("application_errors_total", &event.MetricOptions{
		Description: "Total application errors",
	})

	// Simulate various error scenarios
	errorScenarios := []struct {
		name      string
		errorType string
	}{
		{"database_connection_timeout", "database"},
		{"external_api_rate_limit", "external_api"},
		{"invalid_user_input", "validation"},
		{"payment_gateway_error", "payment"},
	}

	for _, scenario := range errorScenarios {
		simulateErrorScenario(ctx, errorCounter, scenario.name, scenario.errorType)
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("Slog errors captured: %d\n", len(slogErrors))
	fmt.Printf("Prometheus errors captured: %d\n", len(prometheusErrors))
	fmt.Println("✅ Error handling example completed")
}

func simulateErrorScenario(ctx context.Context, errorCounter *event.Counter, errorName, errorType string) {
	event.Log(ctx, "error scenario occurred",
		event.String("error_name", errorName),
		event.String("error_type", errorType),
		event.String("severity", "high"),
	)

	errorCounter.Record(ctx, 1,
		event.String("error_type", errorType),
		event.String("severity", "high"),
	)

	// Log recovery attempt
	event.Log(ctx, "attempting error recovery",
		event.String("error_name", errorName),
		event.String("recovery_strategy", "retry_with_backoff"),
	)

	// Simulate recovery time
	time.Sleep(50 * time.Millisecond)

	event.Log(ctx, "error recovery completed",
		event.String("error_name", errorName),
		event.String("status", "recovered"),
	)
}

// Example 4: Performance Monitoring
func runPerformanceMonitoringExample() {
	fmt.Println("\n=== Performance Monitoring Example ===")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	slogHandler, _ := telemetry.NewSlogHandler(logger)
	prometheusReg := prometheus.NewRegistry()
	prometheusHandler, _ := telemetry.NewPrometheusHandler(prometheusReg)

	multiHandler := &telemetry.MultiHandler{
		Log:    slogHandler,
		Metric: prometheusHandler,
	}

	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(multiHandler, eventtest.ExporterOptions()))

	fmt.Println("Monitoring performance of various operations...")

	// Create performance metrics
	queryDuration := event.NewDuration("database_query_duration", &event.MetricOptions{
		Description: "Database query execution time",
	})

	cacheHits := event.NewCounter("cache_operations_total", &event.MetricOptions{
		Description: "Cache operations",
	})

	// Simulate database operations with varying performance
	queries := []struct {
		query            string
		complexity       string
		expectedDuration time.Duration
	}{
		{"SELECT * FROM users WHERE id = ?", "simple", 10 * time.Millisecond},
		{"SELECT * FROM orders JOIN users ON orders.user_id = users.id", "medium", 50 * time.Millisecond},
		{"SELECT COUNT(*) FROM transactions WHERE date > ? GROUP BY user_id", "complex", 200 * time.Millisecond},
	}

	for i, query := range queries {
		for j := 0; j < 3; j++ {
			simulateQuery(ctx, queryDuration, cacheHits, query.query, query.complexity, query.expectedDuration)
			time.Sleep(100 * time.Millisecond)
		}

		if i < len(queries)-1 {
			fmt.Printf("Completed %s queries, moving to next complexity level...\n", query.complexity)
		}
	}

	fmt.Println("✅ Performance monitoring example completed")
}

func simulateQuery(ctx context.Context, queryDuration *event.DurationDistribution, cacheHits *event.Counter, query, complexity string, expectedDuration time.Duration) {
	start := time.Now()

	event.Log(ctx, "executing database query",
		event.String("query_type", complexity),
		event.String("query", query[:min(50, len(query))]+"..."),
	)

	// Simulate cache check
	cacheHit := rand.Float64() < 0.3 // 30% cache hit rate
	if cacheHit {
		cacheHits.Record(ctx, 1,
			event.String("operation", "hit"),
			event.String("query_type", complexity),
		)

		event.Log(ctx, "cache hit",
			event.String("query_type", complexity),
		)

		// Cache hits are much faster
		time.Sleep(2 * time.Millisecond)
	} else {
		cacheHits.Record(ctx, 1,
			event.String("operation", "miss"),
			event.String("query_type", complexity),
		)

		// Simulate actual database query with some variance
		variance := time.Duration(rand.Intn(int(expectedDuration.Nanoseconds()/2))) * time.Nanosecond
		actualDuration := expectedDuration + variance
		time.Sleep(actualDuration)
	}

	duration := time.Since(start)
	queryDuration.Record(ctx, duration,
		event.String("query_type", complexity),
		event.Bool("cache_hit", cacheHit),
	)

	event.Log(ctx, "query execution completed",
		event.String("query_type", complexity),
		event.Bool("cache_hit", cacheHit),
		event.Int64("duration_ms", duration.Milliseconds()),
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Example 5: Web Server with Prometheus Metrics Endpoint
func runWebServerExample() {
	fmt.Println("\n=== Web Server with Metrics Example ===")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	slogHandler, _ := telemetry.NewSlogHandler(logger)
	prometheusReg := prometheus.NewRegistry()
	prometheusHandler, _ := telemetry.NewPrometheusHandler(prometheusReg)

	multiHandler := &telemetry.MultiHandler{
		Log:    slogHandler,
		Metric: prometheusHandler,
	}

	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(multiHandler, eventtest.ExporterOptions()))

	// Create HTTP server metrics
	requestCounter := event.NewCounter("web_requests_total", &event.MetricOptions{
		Description: "Total web requests",
	})

	requestDuration := event.NewDuration("web_request_duration", &event.MetricOptions{
		Description: "Web request duration",
	})

	// Setup HTTP server
	mux := http.NewServeMux()

	// Add a simple endpoint that generates telemetry
	mux.HandleFunc("/api/demo", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		event.Log(ctx, "demo endpoint called",
			event.String("method", r.Method),
			event.String("path", r.URL.Path),
		)

		// Simulate some work
		time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)

		requestCounter.Record(ctx, 1,
			event.String("method", r.Method),
			event.String("endpoint", "/api/demo"),
			event.String("status", "200"),
		)

		requestDuration.Record(ctx, time.Since(start),
			event.String("method", r.Method),
			event.String("endpoint", "/api/demo"),
		)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message": "Demo endpoint", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))

		event.Log(ctx, "demo endpoint response sent",
			event.String("method", r.Method),
			event.Int64("duration_ms", time.Since(start).Milliseconds()),
		)
	})

	// Add Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.HandlerFor(prometheusReg, promhttp.HandlerOpts{}))

	// Add status endpoint
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status": "ok", "service": "telemetry-demo"}`)
	})

	// Simulate some requests for demonstration
	fmt.Println("Simulating web requests...")

	for i := 0; i < 3; i++ {
		// Simulate processing a request (in real server this would be handled by HTTP requests)
		start := time.Now()

		event.Log(ctx, "simulated web request",
			event.String("method", "GET"),
			event.String("path", "/api/demo"),
			event.Int64("request_id", int64(i+1)),
		)

		// Simulate request processing
		time.Sleep(time.Duration(30+rand.Intn(70)) * time.Millisecond)

		requestCounter.Record(ctx, 1,
			event.String("method", "GET"),
			event.String("endpoint", "/api/demo"),
			event.String("status", "200"),
		)

		requestDuration.Record(ctx, time.Since(start),
			event.String("method", "GET"),
			event.String("endpoint", "/api/demo"),
		)

		event.Log(ctx, "simulated request completed",
			event.Int64("request_id", int64(i+1)),
			event.Int64("duration_ms", time.Since(start).Milliseconds()),
		)

		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println("✅ Web server example completed")
	fmt.Println("💡 In a real application, start the server with:")
	fmt.Println("   go run examples/real_world_examples.go")
	fmt.Println("   Then visit http://localhost:8080/metrics to see Prometheus metrics")
}

/*
func main() {
	fmt.Println("🚀 Telemetry Package - Real World Examples")
	fmt.Println("==========================================")

	// Run all examples
	runBasicWebExample()
	runBusinessLogicExample()
	runErrorHandlingExample()
	runPerformanceMonitoringExample()
	runWebServerExample()

	fmt.Println("\n🎉 All examples completed successfully!")
	fmt.Println("\nThese examples demonstrate:")
	fmt.Println("✅ Structured logging with contextual information")
	fmt.Println("✅ Business metrics collection and reporting")
	fmt.Println("✅ Error handling and recovery monitoring")
	fmt.Println("✅ Performance monitoring and optimization")
	fmt.Println("✅ Integration with Prometheus for metrics export")
	fmt.Println("✅ Real-world usage patterns and best practices")
}
*/
