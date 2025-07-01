package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/alextanhongpin/core/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// responseWriterWithStatus wraps http.ResponseWriter to capture status code
type responseWriterWithStatus struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWithStatus) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterWithStatus) Write(b []byte) (int, error) {
	// If WriteHeader wasn't called explicitly, default to 200
	if rw.statusCode == 0 {
		rw.statusCode = 200
	}
	return rw.ResponseWriter.Write(b)
}

// ProductService demonstrates real-world usage with proper error handling
type ProductService struct {
	tracker   *metrics.Tracker
	logger    *slog.Logger
	products  map[int]Product
	analytics *metrics.Tracker
}

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	TraceID string `json:"trace_id,omitempty"`
}

func NewProductService(redisClient *redis.Client, logger *slog.Logger) *ProductService {
	// Initialize with sample data
	products := map[int]Product{
		1: {ID: 1, Name: "Laptop", Price: 999.99},
		2: {ID: 2, Name: "Mouse", Price: 29.99},
		3: {ID: 3, Name: "Keyboard", Price: 79.99},
	}

	// Provide a default logger if none is provided
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	tracker := metrics.NewTracker("product_service", redisClient)

	return &ProductService{
		tracker:   tracker,
		logger:    logger,
		products:  products,
		analytics: tracker,
	}
}

// GetProduct demonstrates comprehensive error handling and metrics
func (s *ProductService) GetProduct(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Wrap the response writer to capture status code
	rw := &responseWriterWithStatus{ResponseWriter: w, statusCode: 200}

	// Initialize RED metrics tracking
	red := metrics.NewRED("product_service", "get_product")
	defer func() {
		// Record request duration with actual status code
		status := fmt.Sprintf("%d", rw.statusCode)

		metrics.RequestDuration.
			WithLabelValues(r.Method, "/api/product", status, "v1").
			Observe(float64(time.Since(start).Seconds()))

		// Ensure metrics are always recorded, even on panic
		if r := recover(); r != nil {
			red.SetStatus("panic")
			red.Done()
			s.logger.Error("panic in GetProduct", "error", r)
			s.writeErrorResponse(rw, "Internal server error", "INTERNAL_ERROR", http.StatusInternalServerError)
			return
		}
		red.Done()
	}()

	// Use the wrapped writer for the rest of the function
	w = rw

	// Track in-flight requests
	metrics.InFlightGauge.Inc()
	defer metrics.InFlightGauge.Dec()

	// Extract and validate product ID
	productIDStr := r.URL.Query().Get("id")
	if productIDStr == "" {
		red.SetStatus("missing_id")
		s.writeErrorResponse(w, "Product ID is required", "MISSING_ID", http.StatusBadRequest)
		return
	}

	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		red.SetStatus("invalid_id")
		s.logger.Warn("invalid product ID", "id", productIDStr, "error", err)
		s.writeErrorResponse(w, "Invalid product ID", "INVALID_ID", http.StatusBadRequest)
		return
	}

	// Simulate occasional service errors (before database lookup)
	if productID == 500 {
		red.SetStatus("service_error")
		s.logger.Error("simulated service error", "id", productID)
		s.writeErrorResponse(w, "Service temporarily unavailable", "SERVICE_ERROR", http.StatusServiceUnavailable)
		return
	}

	// Simulate database lookup with potential failures
	product, exists := s.findProduct(productID)
	if !exists {
		red.SetStatus("not_found")
		s.logger.Info("product not found", "id", productID)
		s.writeErrorResponse(w, "Product not found", "NOT_FOUND", http.StatusNotFound)
		return
	}

	// Success case
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(product); err != nil {
		red.SetStatus("encoding_error")
		s.logger.Error("failed to encode response", "error", err)
		return
	}

	// Track response size (estimate based on JSON encoding)
	responseSize := metrics.ObserveResponseSize(r)
	metrics.ResponseSize.WithLabelValues().Observe(float64(responseSize))
	s.logger.Debug("product retrieved successfully", "id", productID, "name", product.Name)
}

func (s *ProductService) findProduct(id int) (Product, bool) {
	// Simulate database latency
	time.Sleep(10 * time.Millisecond)

	product, exists := s.products[id]
	return product, exists
}

// ListProducts demonstrates bulk operations with analytics
func (s *ProductService) ListProducts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Wrap the response writer to capture status code
	rw := &responseWriterWithStatus{ResponseWriter: w, statusCode: 200}

	red := metrics.NewRED("product_service", "list_products")
	defer func() {
		// Record request duration with actual status code
		status := fmt.Sprintf("%d", rw.statusCode)
		metrics.RequestDuration.
			WithLabelValues(r.Method, "/api/products", status, "v1").
			Observe(float64(time.Since(start).Seconds()))
		red.Done()
	}()

	// Use the wrapped writer for the rest of the function
	w = rw

	metrics.InFlightGauge.Inc()
	defer metrics.InFlightGauge.Dec()

	// Simulate processing time
	time.Sleep(25 * time.Millisecond)

	// Convert map to slice for JSON response
	products := make([]Product, 0, len(s.products))
	for _, product := range s.products {
		products = append(products, product)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(products); err != nil {
		red.SetStatus("encoding_error")
		s.logger.Error("failed to encode products", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Track response size
	responseSize := metrics.ObserveResponseSize(r)
	metrics.ResponseSize.WithLabelValues().Observe(float64(responseSize))
	s.logger.Debug("products listed successfully", "count", len(products))
}

// CreateProduct demonstrates POST operations with validation
func (s *ProductService) CreateProduct(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Wrap the response writer to capture status code
	rw := &responseWriterWithStatus{ResponseWriter: w, statusCode: 200}

	red := metrics.NewRED("product_service", "create_product")
	defer func() {
		// Record request duration with actual status code
		status := fmt.Sprintf("%d", rw.statusCode)
		metrics.RequestDuration.
			WithLabelValues(r.Method, "/api/products/create", status, "v1").
			Observe(float64(time.Since(start).Seconds()))
		red.Done()
	}()

	// Use the wrapped writer for the rest of the function
	w = rw

	metrics.InFlightGauge.Inc()
	defer metrics.InFlightGauge.Dec()

	// Parse request body
	var product Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		red.SetStatus("invalid_json")
		s.writeErrorResponse(w, "Invalid JSON", "INVALID_JSON", http.StatusBadRequest)
		return
	}

	// Validate product data
	if product.Name == "" {
		red.SetStatus("validation_failed")
		s.writeErrorResponse(w, "Product name is required", "VALIDATION_ERROR", http.StatusBadRequest)
		return
	}

	if product.Price <= 0 {
		red.SetStatus("validation_failed")
		s.writeErrorResponse(w, "Product price must be positive", "VALIDATION_ERROR", http.StatusBadRequest)
		return
	}

	// Generate ID and simulate database save
	product.ID = len(s.products) + 1
	s.products[product.ID] = product

	// Simulate database latency
	time.Sleep(50 * time.Millisecond)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(product); err != nil {
		red.SetStatus("encoding_error")
		s.logger.Error("failed to encode created product", "error", err)
		return
	}

	// Track response size
	responseSize := metrics.ObserveResponseSize(r)
	metrics.ResponseSize.WithLabelValues().Observe(float64(responseSize))
	s.logger.Info("product created successfully", "id", product.ID, "name", product.Name)
}

func (s *ProductService) writeErrorResponse(w http.ResponseWriter, message, code string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error: message,
		Code:  code,
	}

	json.NewEncoder(w).Encode(response)

	// Record response size for error responses (estimated)
	respSize := len(message) + len(code) + 50 // JSON overhead
	metrics.ResponseSize.WithLabelValues().Observe(float64(respSize))
}

// Health check endpoint with metrics
func (s *ProductService) HealthCheck(w http.ResponseWriter, r *http.Request) {
	red := metrics.NewRED("product_service", "health_check")
	defer red.Done()

	// Simple health check - could include database ping, etc.
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// StatsHandler provides analytics endpoint
func (s *ProductService) StatsHandler(w http.ResponseWriter, r *http.Request) {
	red := metrics.NewRED("product_service", "get_stats")
	defer red.Done()

	// Use the tracker's built-in stats handler
	metrics.TrackerStatsHandler(s.analytics).ServeHTTP(w, r)
}

func setupPrometheusRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()

	// Register application metrics
	registry.MustRegister(
		metrics.InFlightGauge,
		metrics.RequestDuration,
		metrics.ResponseSize,
		metrics.RED,
		// Add Go runtime metrics
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	return registry
}

func setupRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		DB:           1, // Use dedicated DB for metrics
		PoolSize:     10,
		MinIdleConns: 2,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	})
}

func userExtractor(r *http.Request) string {
	// Extract user ID from various sources
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return userID
	}
	if userID := r.URL.Query().Get("user_id"); userID != "" {
		return userID
	}
	// Fallback to IP address
	return r.RemoteAddr
}

func main() {
	// Setup logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Setup Redis client
	redisClient := setupRedisClient()
	defer redisClient.Close()

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn("Redis connection failed, analytics will be limited", "error", err)
	}

	// Setup Prometheus registry
	registry := setupPrometheusRegistry()

	// Create service
	productService := NewProductService(redisClient, logger)

	// Setup HTTP server with comprehensive middleware
	mux := http.NewServeMux()

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))

	// API endpoints with full instrumentation
	apiVersion := "v1.0.0"

	// GET /api/products - list all products
	listHandler := metrics.RequestDurationHandler(apiVersion,
		metrics.TrackerHandler(
			http.HandlerFunc(productService.ListProducts),
			productService.tracker,
			userExtractor,
			logger,
		),
	)
	mux.Handle("/api/products", listHandler)

	// GET /api/product?id=1 - get specific product
	getHandler := metrics.RequestDurationHandler(apiVersion,
		metrics.TrackerHandler(
			http.HandlerFunc(productService.GetProduct),
			productService.tracker,
			userExtractor,
			logger,
		),
	)
	mux.Handle("/api/product", getHandler)

	// POST /api/products - create new product
	createHandler := metrics.RequestDurationHandler(apiVersion,
		metrics.TrackerHandler(
			http.HandlerFunc(productService.CreateProduct),
			productService.tracker,
			userExtractor,
			logger,
		),
	)
	mux.Handle("/api/products/create", createHandler)

	// Health check endpoint
	healthHandler := metrics.RequestDurationHandler(apiVersion,
		http.HandlerFunc(productService.HealthCheck),
	)
	mux.Handle("/health", healthHandler)

	// Analytics endpoint
	mux.Handle("/admin/stats", http.HandlerFunc(productService.StatsHandler))

	// Create server with timeouts
	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("Starting server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	} else {
		logger.Info("Server exited gracefully")
	}
}
