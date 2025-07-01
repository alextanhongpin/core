// Real-world example: E-commerce Order Processing Service
// This example demonstrates comprehensive telemetry in a microservice that processes orders
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/alextanhongpin/core/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/slog"
)

// Order represents an e-commerce order
type Order struct {
	ID          string    `json:"id"`
	CustomerID  string    `json:"customer_id"`
	ProductID   string    `json:"product_id"`
	Quantity    int       `json:"quantity"`
	Amount      float64   `json:"amount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	ProcessedAt time.Time `json:"processed_at,omitempty"`
}

// OrderService handles order processing with comprehensive telemetry
type OrderService struct {
	ctx       context.Context
	telemetry *telemetry.MultiHandler
	orders    map[string]*Order // In-memory store for demo
}

func NewOrderService() (*OrderService, error) {
	// Setup structured logging with custom formatting
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339Nano))
			}
			if a.Key == slog.LevelKey {
				return slog.String("severity", a.Value.String())
			}
			return a
		},
	}))

	slogHandler, err := telemetry.NewSlogHandler(logger, telemetry.WithSlogErrorHandler(func(err error) {
		log.Printf("SlogHandler error: %v", err)
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create slog handler: %w", err)
	}

	// Setup Prometheus metrics
	reg := prometheus.NewRegistry()
	prometheusHandler, err := telemetry.NewPrometheusHandler(reg, telemetry.WithPrometheusErrorHandler(func(err error) {
		log.Printf("PrometheusHandler error: %v", err)
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus handler: %w", err)
	}

	// Setup OpenTelemetry metrics
	meter := otel.Meter("order-service",
		"metric.WithInstrumentationVersion", "1.0.0",
	)

	metricHandler, err := telemetry.NewMetricHandler(meter, telemetry.WithErrorHandler(func(err error) {
		log.Printf("MetricHandler error: %v", err)
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create metric handler: %w", err)
	}

	// Create unified telemetry handler
	multiHandler := &telemetry.MultiHandler{
		Log:    slogHandler,
		Metric: metricHandler,
		Trace:  prometheusHandler, // Using prometheus as trace handler for demo
	}

	// Setup event context
	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(multiHandler, eventtest.ExporterOptions()))

	// Register custom metrics endpoint
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	service := &OrderService{
		ctx:       ctx,
		telemetry: multiHandler,
		orders:    make(map[string]*Order),
	}

	// Log service initialization
	event.Log(ctx, "order service initialized",
		event.String("component", "order-service"),
		event.String("version", "1.0.0"),
		event.String("environment", getEnv("ENV", "development")),
	)

	return service, nil
}

// CreateOrder handles order creation with full telemetry
func (os *OrderService) CreateOrder(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := os.ctx

	// Extract request ID for correlation
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = generateID()
	}

	// Create enriched context for this request
	ctx = event.NewContext(ctx, os.telemetry)

	// Log incoming request
	event.Log(ctx, "processing create order request",
		event.String("request_id", requestID),
		event.String("method", r.Method),
		event.String("path", r.URL.Path),
		event.String("user_agent", r.UserAgent()),
		event.String("remote_addr", r.RemoteAddr),
	)

	// Parse request body
	var orderReq struct {
		CustomerID string  `json:"customer_id"`
		ProductID  string  `json:"product_id"`
		Quantity   int     `json:"quantity"`
		Amount     float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&orderReq); err != nil {
		os.handleError(ctx, w, "invalid request body", err, requestID, http.StatusBadRequest)
		return
	}

	// Validate request
	if err := os.validateOrderRequest(orderReq); err != nil {
		os.handleError(ctx, w, "validation failed", err, requestID, http.StatusBadRequest)
		return
	}

	// Create order
	order := &Order{
		ID:         generateID(),
		CustomerID: orderReq.CustomerID,
		ProductID:  orderReq.ProductID,
		Quantity:   orderReq.Quantity,
		Amount:     orderReq.Amount,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	// Simulate business logic with telemetry
	if err := os.processOrder(ctx, order, requestID); err != nil {
		os.handleError(ctx, w, "order processing failed", err, requestID, http.StatusInternalServerError)
		return
	}

	// Store order
	os.orders[order.ID] = order

	// Record success metrics
	orderCounter := event.NewCounter("orders_created_total", &event.MetricOptions{
		Description: "Total orders created",
	})
	orderCounter.Record(ctx, 1,
		event.String("status", "success"),
		event.String("product_id", order.ProductID),
		event.String("customer_id", order.CustomerID),
	)

	processingTime := event.NewDuration("order_processing_duration", &event.MetricOptions{
		Description: "Order processing duration",
		Unit:        event.UnitMilliseconds,
	})
	processingTime.Record(ctx, time.Since(start),
		event.String("status", "success"),
	)

	// Log successful creation
	event.Log(ctx, "order created successfully",
		event.String("request_id", requestID),
		event.String("order_id", order.ID),
		event.String("customer_id", order.CustomerID),
		event.String("product_id", order.ProductID),
		event.Int64("quantity", int64(order.Quantity)),
		event.Float64("amount", order.Amount),
		event.Int64("processing_time_ms", time.Since(start).Milliseconds()),
	)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

// GetOrder retrieves an order by ID
func (os *OrderService) GetOrder(w http.ResponseWriter, r *http.Request) {
	ctx := os.ctx
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = generateID()
	}

	orderID := r.URL.Query().Get("id")
	if orderID == "" {
		os.handleError(ctx, w, "missing order ID", fmt.Errorf("order_id parameter required"), requestID, http.StatusBadRequest)
		return
	}

	event.Log(ctx, "retrieving order",
		event.String("request_id", requestID),
		event.String("order_id", orderID),
	)

	// Simulate database lookup
	time.Sleep(10 * time.Millisecond)

	order, exists := os.orders[orderID]
	if !exists {
		event.Log(ctx, "order not found",
			event.String("request_id", requestID),
			event.String("order_id", orderID),
		)

		// Record not found metric
		notFoundCounter := event.NewCounter("orders_not_found_total", &event.MetricOptions{
			Description: "Total orders not found",
		})
		notFoundCounter.Record(ctx, 1)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", requestID)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "order not found"})
		return
	}

	// Record retrieval metric
	retrievalCounter := event.NewCounter("orders_retrieved_total", &event.MetricOptions{
		Description: "Total orders retrieved",
	})
	retrievalCounter.Record(ctx, 1,
		event.String("status", order.Status),
	)

	event.Log(ctx, "order retrieved successfully",
		event.String("request_id", requestID),
		event.String("order_id", orderID),
		event.String("status", order.Status),
	)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	json.NewEncoder(w).Encode(order)
}

// processOrder simulates complex business logic with telemetry
func (os *OrderService) processOrder(ctx context.Context, order *Order, requestID string) error {
	event.Log(ctx, "starting order processing",
		event.String("request_id", requestID),
		event.String("order_id", order.ID),
		event.String("phase", "validation"),
	)

	// Simulate inventory check
	if err := os.checkInventory(ctx, order.ProductID, order.Quantity, requestID); err != nil {
		return fmt.Errorf("inventory check failed: %w", err)
	}

	// Simulate payment processing
	if err := os.processPayment(ctx, order.CustomerID, order.Amount, requestID); err != nil {
		return fmt.Errorf("payment processing failed: %w", err)
	}

	// Simulate fulfillment
	if err := os.scheduleFulfillment(ctx, order, requestID); err != nil {
		return fmt.Errorf("fulfillment scheduling failed: %w", err)
	}

	order.Status = "confirmed"
	order.ProcessedAt = time.Now()

	event.Log(ctx, "order processing completed",
		event.String("request_id", requestID),
		event.String("order_id", order.ID),
		event.String("final_status", order.Status),
	)

	return nil
}

func (os *OrderService) checkInventory(ctx context.Context, productID string, quantity int, requestID string) error {
	start := time.Now()

	event.Log(ctx, "checking inventory",
		event.String("request_id", requestID),
		event.String("product_id", productID),
		event.Int64("requested_quantity", int64(quantity)),
	)

	// Simulate external service call
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

	// Simulate occasional inventory shortage (10% chance)
	if rand.Float32() < 0.1 {
		event.Log(ctx, "insufficient inventory",
			event.String("request_id", requestID),
			event.String("product_id", productID),
			event.Int64("requested_quantity", int64(quantity)),
			event.String("error", "insufficient_stock"),
		)

		inventoryCounter := event.NewCounter("inventory_checks_total", &event.MetricOptions{
			Description: "Total inventory checks",
		})
		inventoryCounter.Record(ctx, 1,
			event.String("result", "insufficient"),
			event.String("product_id", productID),
		)

		return fmt.Errorf("insufficient inventory for product %s", productID)
	}

	// Record successful inventory check
	inventoryCounter := event.NewCounter("inventory_checks_total", &event.MetricOptions{
		Description: "Total inventory checks",
	})
	inventoryCounter.Record(ctx, 1,
		event.String("result", "success"),
		event.String("product_id", productID),
	)

	inventoryLatency := event.NewDuration("inventory_check_duration", &event.MetricOptions{
		Description: "Inventory check duration",
	})
	inventoryLatency.Record(ctx, time.Since(start))

	event.Log(ctx, "inventory check completed",
		event.String("request_id", requestID),
		event.String("product_id", productID),
		event.Int64("available_quantity", int64(quantity+10)), // Mock available quantity
		event.Int64("duration_ms", time.Since(start).Milliseconds()),
	)

	return nil
}

func (os *OrderService) processPayment(ctx context.Context, customerID string, amount float64, requestID string) error {
	start := time.Now()

	event.Log(ctx, "processing payment",
		event.String("request_id", requestID),
		event.String("customer_id", customerID),
		event.Float64("amount", amount),
	)

	// Simulate payment gateway call
	time.Sleep(time.Duration(rand.Intn(100)+50) * time.Millisecond)

	// Simulate payment failure (5% chance)
	if rand.Float32() < 0.05 {
		event.Log(ctx, "payment failed",
			event.String("request_id", requestID),
			event.String("customer_id", customerID),
			event.Float64("amount", amount),
			event.String("error", "payment_declined"),
		)

		paymentCounter := event.NewCounter("payments_total", &event.MetricOptions{
			Description: "Total payment attempts",
		})
		paymentCounter.Record(ctx, 1,
			event.String("result", "failed"),
			event.String("reason", "declined"),
		)

		return fmt.Errorf("payment declined for customer %s", customerID)
	}

	// Record successful payment
	paymentCounter := event.NewCounter("payments_total", &event.MetricOptions{
		Description: "Total payment attempts",
	})
	paymentCounter.Record(ctx, 1,
		event.String("result", "success"),
	)

	paymentAmountHistogram := event.NewDuration("payment_amount", &event.MetricOptions{
		Description: "Payment amounts processed",
	})
	paymentAmountHistogram.Record(ctx, time.Duration(amount*1000)) // Convert to duration for demo

	event.Log(ctx, "payment processed successfully",
		event.String("request_id", requestID),
		event.String("customer_id", customerID),
		event.Float64("amount", amount),
		event.String("payment_id", generateID()),
		event.Int64("duration_ms", time.Since(start).Milliseconds()),
	)

	return nil
}

func (os *OrderService) scheduleFulfillment(ctx context.Context, order *Order, requestID string) error {
	event.Log(ctx, "scheduling fulfillment",
		event.String("request_id", requestID),
		event.String("order_id", order.ID),
		event.String("product_id", order.ProductID),
	)

	// Simulate fulfillment scheduling
	time.Sleep(20 * time.Millisecond)

	fulfillmentCounter := event.NewCounter("fulfillments_scheduled_total", &event.MetricOptions{
		Description: "Total fulfillments scheduled",
	})
	fulfillmentCounter.Record(ctx, 1,
		event.String("product_id", order.ProductID),
	)

	estimatedDelivery := time.Now().Add(48 * time.Hour)

	event.Log(ctx, "fulfillment scheduled",
		event.String("request_id", requestID),
		event.String("order_id", order.ID),
		event.String("estimated_delivery", estimatedDelivery.Format(time.RFC3339)),
		event.String("fulfillment_center", "warehouse-east-1"),
	)

	return nil
}

func (os *OrderService) handleError(ctx context.Context, w http.ResponseWriter, message string, err error, requestID string, statusCode int) {
	event.Log(ctx, message,
		event.String("request_id", requestID),
		event.String("error", err.Error()),
		event.Int64("status_code", int64(statusCode)),
	)

	// Record error metric
	errorCounter := event.NewCounter("http_errors_total", &event.MetricOptions{
		Description: "Total HTTP errors",
	})
	errorCounter.Record(ctx, 1,
		event.String("error_type", message),
		event.Int64("status_code", int64(statusCode)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":      message,
		"details":    err.Error(),
		"request_id": requestID,
		"timestamp":  time.Now().Format(time.RFC3339),
	})
}

func (os *OrderService) validateOrderRequest(req struct {
	CustomerID string  `json:"customer_id"`
	ProductID  string  `json:"product_id"`
	Quantity   int     `json:"quantity"`
	Amount     float64 `json:"amount"`
}) error {
	if req.CustomerID == "" {
		return fmt.Errorf("customer_id is required")
	}
	if req.ProductID == "" {
		return fmt.Errorf("product_id is required")
	}
	if req.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

// Health check endpoint with telemetry
func (os *OrderService) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := os.ctx
	start := time.Now()

	event.Log(ctx, "health check requested")

	// Simulate health checks for dependencies
	healthStatus := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
		"checks": map[string]interface{}{
			"database":          os.checkDatabaseHealth(ctx),
			"payment_gateway":   os.checkPaymentGatewayHealth(ctx),
			"inventory_service": os.checkInventoryServiceHealth(ctx),
		},
		"uptime_seconds": time.Since(start).Seconds(),
	}

	// Determine overall health
	allHealthy := true
	checks := healthStatus["checks"].(map[string]interface{})
	for service, status := range checks {
		if status.(map[string]interface{})["healthy"].(bool) == false {
			allHealthy = false
			event.Log(ctx, "unhealthy dependency detected",
				event.String("service", service),
			)
		}
	}

	if !allHealthy {
		healthStatus["status"] = "unhealthy"
	}

	// Record health check metric
	healthCounter := event.NewCounter("health_checks_total", &event.MetricOptions{
		Description: "Total health checks",
	})
	status := "healthy"
	if !allHealthy {
		status = "unhealthy"
	}
	healthCounter.Record(ctx, 1,
		event.String("status", status),
	)

	event.Log(ctx, "health check completed",
		event.String("overall_status", status),
		event.Int64("duration_ms", time.Since(start).Milliseconds()),
	)

	w.Header().Set("Content-Type", "application/json")
	if allHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(healthStatus)
}

func (os *OrderService) checkDatabaseHealth(ctx context.Context) map[string]interface{} {
	// Simulate database ping
	time.Sleep(5 * time.Millisecond)
	return map[string]interface{}{
		"healthy":          true,
		"response_time_ms": 5,
		"last_check":       time.Now().Format(time.RFC3339),
	}
}

func (os *OrderService) checkPaymentGatewayHealth(ctx context.Context) map[string]interface{} {
	// Simulate payment gateway health check
	time.Sleep(10 * time.Millisecond)
	return map[string]interface{}{
		"healthy":          true,
		"response_time_ms": 10,
		"last_check":       time.Now().Format(time.RFC3339),
	}
}

func (os *OrderService) checkInventoryServiceHealth(ctx context.Context) map[string]interface{} {
	// Simulate inventory service health check
	time.Sleep(8 * time.Millisecond)
	return map[string]interface{}{
		"healthy":          true,
		"response_time_ms": 8,
		"last_check":       time.Now().Format(time.RFC3339),
	}
}

// Close cleans up resources
func (os *OrderService) Close() error {
	event.Log(os.ctx, "shutting down order service")
	if os.telemetry != nil {
		return os.telemetry.Close()
	}
	return nil
}

// Utility functions
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func runOrderService() {
	service, err := NewOrderService()
	if err != nil {
		log.Fatalf("Failed to create order service: %v", err)
	}
	defer service.Close()

	// Setup routes
	http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			service.CreateOrder(w, r)
		case http.MethodGet:
			service.GetOrder(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/health", service.HealthCheck)

	// Log startup
	event.Log(service.ctx, "order service starting",
		event.String("port", "8081"),
		event.String("version", "1.0.0"),
		event.String("environment", getEnv("ENV", "development")),
	)

	log.Println("Order Service starting on :8081")
	log.Println("Endpoints:")
	log.Println("  POST /orders - Create a new order")
	log.Println("  GET /orders?id=<order_id> - Get order by ID")
	log.Println("  GET /health - Health check")
	log.Println("  GET /metrics - Prometheus metrics")
	log.Println("")
	log.Println("Example request:")
	log.Println(`curl -X POST http://localhost:8081/orders \`)
	log.Println(`  -H "Content-Type: application/json" \`)
	log.Println(`  -H "X-Request-ID: test-123" \`)
	log.Println(`  -d '{"customer_id":"cust-123","product_id":"prod-456","quantity":2,"amount":29.99}'`)

	if err := http.ListenAndServe(":8081", nil); err != nil {
		event.Log(service.ctx, "server error",
			event.String("error", err.Error()),
		)
		log.Fatalf("Server failed: %v", err)
	}
}

// This would be in a separate file in a real application
func main2() {
	runOrderService()
}
