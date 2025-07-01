package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alextanhongpin/core/telemetry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/slog"
)

// WebService demonstrates a real-world web service with comprehensive telemetry
type WebService struct {
	telemetry *telemetry.MultiHandler
	ctx       context.Context

	// Metrics
	requestCounter    event.Counter
	responseTime      event.Duration
	activeConnections event.Gauge

	// Application state
	connections int
}

func NewWebService() (*WebService, error) {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}))

	slogHandler, err := telemetry.NewSlogHandler(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create slog handler: %w", err)
	}

	// Setup Prometheus metrics
	prometheusReg := prometheus.NewRegistry()
	prometheusHandler, err := telemetry.NewPrometheusHandler(prometheusReg)
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus handler: %w", err)
	}

	// Setup OpenTelemetry metrics
	promExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	meterProvider := metric.NewMeterProvider(metric.WithReader(promExporter))
	otel.SetMeterProvider(meterProvider)

	meter := meterProvider.Meter("webservice",
		metric.WithInstrumentationVersion("1.0.0"),
		metric.WithSchemaURL("https://opentelemetry.io/schemas/1.21.0"),
	)

	metricHandler, err := telemetry.NewMetricHandler(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric handler: %w", err)
	}

	// Create multi-handler for unified telemetry
	multiHandler := &telemetry.MultiHandler{
		Log:    slogHandler,
		Metric: metricHandler,
		//Trace: traceHandler, // Would add tracing handler here
	}

	// Setup event context
	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(multiHandler, eventtest.ExporterOptions()))

	service := &WebService{
		telemetry: multiHandler,
		ctx:       ctx,
	}

	// Initialize metrics
	service.setupMetrics()

	return service, nil
}

func (ws *WebService) setupMetrics() {
	// Define application metrics
	ws.requestCounter = event.NewCounter("http_requests_total", &event.MetricOptions{
		Description: "Total number of HTTP requests",
		Unit:        event.UnitDimensionless,
	})

	ws.responseTime = event.NewDuration("http_request_duration", &event.MetricOptions{
		Description: "HTTP request duration in milliseconds",
		Unit:        event.UnitMilliseconds,
	})

	ws.activeConnections = event.NewGauge("active_connections", &event.MetricOptions{
		Description: "Number of active connections",
		Unit:        event.UnitDimensionless,
	})
}

// Middleware for telemetry
func (ws *WebService) TelemetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Increment connection count
		ws.connections++
		ws.activeConnections.Record(ws.ctx, float64(ws.connections))

		// Create request-scoped context with trace information
		ctx := event.NewContext(r.Context(), ws.telemetry)

		// Log incoming request
		event.Log(ctx, "incoming request",
			event.String("method", r.Method),
			event.String("path", r.URL.Path),
			event.String("remote_addr", r.RemoteAddr),
			event.String("user_agent", r.UserAgent()),
		)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		// Process request
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// Record metrics and logs
		duration := time.Since(start)

		// Record request metrics
		ws.requestCounter.Record(ctx, 1,
			event.String("method", r.Method),
			event.String("path", r.URL.Path),
			event.Int64("status", int64(wrapped.statusCode)),
		)

		ws.responseTime.Record(ctx, duration,
			event.String("method", r.Method),
			event.String("path", r.URL.Path),
		)

		// Log request completion
		level := "info"
		if wrapped.statusCode >= 400 {
			level = "error"
		} else if wrapped.statusCode >= 300 {
			level = "warn"
		}

		event.Log(ctx, "request completed",
			event.String("level", level),
			event.String("method", r.Method),
			event.String("path", r.URL.Path),
			event.Int64("status", int64(wrapped.statusCode)),
			event.Int64("duration_ms", duration.Milliseconds()),
			event.Int64("size", int64(wrapped.size)),
		)

		// Decrement connection count
		ws.connections--
		ws.activeConnections.Record(ws.ctx, float64(ws.connections))
	})
}

// Business logic handlers with telemetry
func (ws *WebService) handleUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		ws.handleGetUsers(ctx, w, r)
	case http.MethodPost:
		ws.handleCreateUser(ctx, w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (ws *WebService) handleGetUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Simulate database operation with telemetry
	start := time.Now()

	event.Log(ctx, "fetching users from database",
		event.String("operation", "db_query"),
		event.String("table", "users"),
	)

	// Simulate database delay and potential error
	time.Sleep(50 * time.Millisecond)

	// Record database operation metrics
	dbDuration := time.Since(start)
	dbCounter := event.NewCounter("db_operations_total", &event.MetricOptions{
		Description: "Total database operations",
	})
	dbLatency := event.NewDuration("db_operation_duration", &event.MetricOptions{
		Description: "Database operation duration",
	})

	dbCounter.Record(ctx, 1,
		event.String("operation", "select"),
		event.String("table", "users"),
		event.String("status", "success"),
	)

	dbLatency.Record(ctx, dbDuration,
		event.String("operation", "select"),
		event.String("table", "users"),
	)

	// Mock user data
	users := []map[string]interface{}{
		{"id": 1, "name": "Alice", "email": "alice@example.com"},
		{"id": 2, "name": "Bob", "email": "bob@example.com"},
	}

	event.Log(ctx, "users retrieved successfully",
		event.Int64("count", int64(len(users))),
		event.Int64("duration_ms", dbDuration.Milliseconds()),
	)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"users": %v, "count": %d}`, users, len(users))
}

func (ws *WebService) handleCreateUser(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Parse request body (simplified)
	name := r.FormValue("name")
	email := r.FormValue("email")

	if name == "" || email == "" {
		event.Log(ctx, "invalid user creation request",
			event.String("error", "missing required fields"),
			event.String("name", name),
			event.String("email", email),
		)
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}

	// Simulate user creation with telemetry
	start := time.Now()

	event.Log(ctx, "creating new user",
		event.String("operation", "user_create"),
		event.String("name", name),
		event.String("email", email),
	)

	// Simulate database operation
	time.Sleep(30 * time.Millisecond)

	// Record business metrics
	userCreationCounter := event.NewCounter("users_created_total", &event.MetricOptions{
		Description: "Total users created",
	})

	userCreationCounter.Record(ctx, 1,
		event.String("status", "success"),
	)

	duration := time.Since(start)

	event.Log(ctx, "user created successfully",
		event.String("operation", "user_create"),
		event.String("name", name),
		event.String("email", email),
		event.Int64("duration_ms", duration.Milliseconds()),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id": 3, "name": "%s", "email": "%s", "created_at": "%s"}`,
		name, email, time.Now().Format(time.RFC3339))
}

func (ws *WebService) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Health check with telemetry
	healthy := true
	components := map[string]bool{
		"database": true, // Would check actual DB connection
		"cache":    true, // Would check cache connection
		"queue":    true, // Would check message queue
	}

	for component, status := range components {
		event.Log(ctx, "health check",
			event.String("component", component),
			event.Bool("healthy", status),
		)
		if !status {
			healthy = false
		}
	}

	if healthy {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status": "healthy", "components": {"database": true, "cache": true, "queue": true}}`)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"status": "unhealthy"}`)
	}

	// Record health check metric
	healthCounter := event.NewCounter("health_checks_total", &event.MetricOptions{
		Description: "Total health checks",
	})

	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}

	healthCounter.Record(ctx, 1,
		event.String("status", status),
	)
}

// Close cleans up resources
func (ws *WebService) Close() error {
	event.Log(ws.ctx, "shutting down web service")
	if ws.telemetry != nil {
		return ws.telemetry.Close()
	}
	return nil
}

// Helper types
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	size, err := w.ResponseWriter.Write(data)
	w.size += size
	return size, err
}

func main() {
	service, err := NewWebService()
	if err != nil {
		log.Fatalf("Failed to create web service: %v", err)
	}
	defer service.Close()

	// Setup routes with telemetry middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/users", service.handleUsers)
	mux.HandleFunc("/health", service.handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

	// Wrap with telemetry middleware
	handler := service.TelemetryMiddleware(mux)

	// Log startup
	event.Log(service.ctx, "starting web service",
		event.String("addr", ":8080"),
		event.String("version", "1.0.0"),
	)

	log.Println("Web service starting on :8080")
	log.Println("Metrics available at http://localhost:8080/metrics")
	log.Println("Health check at http://localhost:8080/health")
	log.Println("Users API at http://localhost:8080/users")

	if err := http.ListenAndServe(":8080", handler); err != nil {
		event.Log(service.ctx, "server error",
			event.String("error", err.Error()),
		)
		log.Fatalf("Server failed: %v", err)
	}
}
