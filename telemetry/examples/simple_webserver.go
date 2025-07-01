package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alextanhongpin/core/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/slog"
)

// runSimpleWebServer demonstrates basic telemetry usage in a web application
func runSimpleWebServer() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}))

	// Create slog handler
	slogHandler, err := telemetry.NewSlogHandler(logger)
	if err != nil {
		log.Fatalf("failed to create slog handler: %v", err)
	}

	// Setup Prometheus metrics
	prometheusReg := prometheus.NewRegistry()
	prometheusHandler, err := telemetry.NewPrometheusHandler(prometheusReg)
	if err != nil {
		log.Fatalf("failed to create prometheus handler: %v", err)
	}

	// Create multi-handler for unified telemetry
	multiHandler := &telemetry.MultiHandler{
		Log:    slogHandler,
		Metric: prometheusHandler,
	}

	// Setup event context
	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(multiHandler, eventtest.ExporterOptions()))

	// Create metrics
	requestCounter := event.NewCounter("http_requests_total", &event.MetricOptions{
		Description: "Total number of HTTP requests",
	})

	requestDuration := event.NewDuration("http_request_duration", &event.MetricOptions{
		Description: "HTTP request duration",
	})

	// Setup HTTP handlers
	mux := http.NewServeMux()

	// Add telemetry middleware
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create request context with telemetry
		reqCtx := r.Context()
		// We can use the global context with the exporter already configured
		reqCtx = ctx

		// Log incoming request
		event.Log(reqCtx, "incoming request",
			event.String("method", r.Method),
			event.String("path", r.URL.Path),
			event.String("remote_addr", r.RemoteAddr),
		)

		// Process the request
		handleHome(reqCtx, w, r)

		// Record metrics
		duration := time.Since(start)
		requestCounter.Record(reqCtx, 1,
			event.String("method", r.Method),
			event.String("path", r.URL.Path),
			event.String("status", "200"),
		)

		requestDuration.Record(reqCtx, duration,
			event.String("method", r.Method),
			event.String("path", r.URL.Path),
		)

		// Log request completion
		event.Log(reqCtx, "request completed",
			event.String("method", r.Method),
			event.String("path", r.URL.Path),
			event.Int64("status", 200),
			event.Int64("duration_ms", duration.Milliseconds()),
		)
	})

	// Add API endpoints
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqCtx := ctx

		event.Log(reqCtx, "API request",
			event.String("endpoint", "/api/users"),
			event.String("method", r.Method),
		)

		// Simulate some business logic
		users := getUsersFromDB(reqCtx)

		duration := time.Since(start)
		requestCounter.Record(reqCtx, 1,
			event.String("method", r.Method),
			event.String("path", "/api/users"),
			event.String("status", "200"),
		)

		requestDuration.Record(reqCtx, duration,
			event.String("method", r.Method),
			event.String("path", "/api/users"),
		)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"users": %v, "count": %d}`, users, len(users))

		event.Log(reqCtx, "API response sent",
			event.String("endpoint", "/api/users"),
			event.Int64("user_count", int64(len(users))),
			event.Int64("duration_ms", duration.Milliseconds()),
		)
	})

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		reqCtx := ctx

		event.Log(reqCtx, "health check",
			event.String("status", "healthy"),
		)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status": "healthy", "timestamp": "`+time.Now().Format(time.RFC3339)+`"}`)
	})

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.HandlerFor(prometheusReg, promhttp.HandlerOpts{}))

	// Start server
	log.Println("Starting server on :8080")
	log.Println("Endpoints:")
	log.Println("  - http://localhost:8080/ (home page)")
	log.Println("  - http://localhost:8080/api/users (user API)")
	log.Println("  - http://localhost:8080/health (health check)")
	log.Println("  - http://localhost:8080/metrics (Prometheus metrics)")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleHome(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	event.Log(ctx, "serving home page")

	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Telemetry Demo</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 800px; margin: 0 auto; }
        .endpoint { background: #f5f5f5; padding: 10px; margin: 10px 0; border-radius: 5px; }
        .method { font-weight: bold; color: #2196F3; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Telemetry Demo Server</h1>
        <p>This server demonstrates real-world usage of the telemetry package with structured logging and Prometheus metrics.</p>
        
        <h2>Available Endpoints:</h2>
        <div class="endpoint">
            <span class="method">GET</span> <a href="/api/users">/api/users</a> - Get list of users
        </div>
        <div class="endpoint">
            <span class="method">GET</span> <a href="/health">/health</a> - Health check
        </div>
        <div class="endpoint">
            <span class="method">GET</span> <a href="/metrics">/metrics</a> - Prometheus metrics
        </div>
        
        <h2>Features Demonstrated:</h2>
        <ul>
            <li>Structured logging with slog</li>
            <li>Prometheus metrics collection</li>
            <li>Request/response tracking</li>
            <li>Business logic telemetry</li>
            <li>Error handling and monitoring</li>
        </ul>
        
        <p>Check the console output to see structured logs, and visit <a href="/metrics">/metrics</a> to see Prometheus metrics.</p>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func getUsersFromDB(ctx context.Context) []map[string]interface{} {
	// Simulate database operation with telemetry
	start := time.Now()

	event.Log(ctx, "database operation started",
		event.String("operation", "SELECT"),
		event.String("table", "users"),
	)

	// Simulate database latency
	time.Sleep(25 * time.Millisecond)

	// Create database operation metrics
	dbCounter := event.NewCounter("db_operations_total", &event.MetricOptions{
		Description: "Total database operations",
	})

	dbDuration := event.NewDuration("db_operation_duration", &event.MetricOptions{
		Description: "Database operation duration",
	})

	// Mock data
	users := []map[string]interface{}{
		{"id": 1, "name": "Alice Johnson", "email": "alice@example.com", "role": "admin"},
		{"id": 2, "name": "Bob Smith", "email": "bob@example.com", "role": "user"},
		{"id": 3, "name": "Carol Davis", "email": "carol@example.com", "role": "user"},
	}

	// Record metrics
	duration := time.Since(start)
	dbCounter.Record(ctx, 1,
		event.String("operation", "SELECT"),
		event.String("table", "users"),
		event.String("status", "success"),
	)

	dbDuration.Record(ctx, duration,
		event.String("operation", "SELECT"),
		event.String("table", "users"),
	)

	event.Log(ctx, "database operation completed",
		event.String("operation", "SELECT"),
		event.String("table", "users"),
		event.Int64("rows_returned", int64(len(users))),
		event.Int64("duration_ms", duration.Milliseconds()),
	)

	return users
}

// func main() {
//	runSimpleWebServer()
// }
