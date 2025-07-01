package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// EcommerceService demonstrates a real-world e-commerce application with comprehensive telemetry
type EcommerceService struct {
	telemetry     *telemetry.MultiHandler
	ctx           context.Context
	prometheusReg *prometheus.Registry

	// Business metrics
	ordersProcessed      *event.Counter
	orderValue           *event.Counter
	processingTime       *event.DurationDistribution
	inventoryChecks      *event.Counter
	paymentAttempts      *event.Counter
	errorRate            *event.Counter
	customerSatisfaction *event.FloatGauge
}

// Business domain models for advanced e-commerce
type EcommerceOrder struct {
	ID          string          `json:"id"`
	CustomerID  string          `json:"customer_id"`
	Items       []EcommerceItem `json:"items"`
	TotalAmount float64         `json:"total_amount"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type EcommerceItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type Customer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Tier  string `json:"tier"` // "standard", "premium", "enterprise"
}

type EcommercePaymentResult struct {
	Success       bool      `json:"success"`
	TransactionID string    `json:"transaction_id,omitempty"`
	Error         string    `json:"error,omitempty"`
	ProcessedAt   time.Time `json:"processed_at"`
}

type InventoryItem struct {
	ProductID string `json:"product_id"`
	Available int    `json:"available"`
	Reserved  int    `json:"reserved"`
}

func NewEcommerceService() (*EcommerceService, error) {
	// Setup structured logging with business context
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}))

	// Setup handlers with error handling
	slogHandler, err := telemetry.NewSlogHandler(logger, telemetry.WithSlogErrorHandler(func(err error) {
		logger.Error("telemetry slog error", "error", err, "component", "slog_handler")
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create slog handler: %w", err)
	}

	prometheusReg := prometheus.NewRegistry()
	prometheusHandler, err := telemetry.NewPrometheusHandler(prometheusReg, telemetry.WithPrometheusErrorHandler(func(err error) {
		logger.Error("telemetry prometheus error", "error", err, "component", "prometheus_handler")
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus handler: %w", err)
	}

	// Create unified telemetry
	multiHandler := &telemetry.MultiHandler{
		Log:    slogHandler,
		Metric: prometheusHandler,
	}

	// Setup telemetry context
	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(multiHandler, eventtest.ExporterOptions()))

	service := &EcommerceService{
		telemetry:     multiHandler,
		ctx:           ctx,
		prometheusReg: prometheusReg,
	}

	service.initializeMetrics()
	return service, nil
}

func (es *EcommerceService) initializeMetrics() {
	// Business-specific metrics that matter for e-commerce
	es.ordersProcessed = event.NewCounter("ecommerce_orders_total", &event.MetricOptions{
		Description: "Total number of orders processed",
		Namespace:   "ecommerce",
	})

	es.orderValue = event.NewCounter("ecommerce_order_value_total", &event.MetricOptions{
		Description: "Total monetary value of orders",
		Unit:        "dollars",
		Namespace:   "ecommerce",
	})

	es.processingTime = event.NewDuration("ecommerce_order_processing_duration", &event.MetricOptions{
		Description: "Time taken to process orders end-to-end",
		Namespace:   "ecommerce",
	})

	es.inventoryChecks = event.NewCounter("ecommerce_inventory_checks_total", &event.MetricOptions{
		Description: "Number of inventory availability checks",
		Namespace:   "ecommerce",
	})

	es.paymentAttempts = event.NewCounter("ecommerce_payment_attempts_total", &event.MetricOptions{
		Description: "Payment processing attempts",
		Namespace:   "ecommerce",
	})

	es.errorRate = event.NewCounter("ecommerce_errors_total", &event.MetricOptions{
		Description: "Business process errors by type",
		Namespace:   "ecommerce",
	})

	es.customerSatisfaction = event.NewFloatGauge("ecommerce_customer_satisfaction_score", &event.MetricOptions{
		Description: "Customer satisfaction score (1-5)",
		Namespace:   "ecommerce",
	})
}

// ProcessOrder demonstrates comprehensive telemetry in a complex business process
func (es *EcommerceService) ProcessOrder(order *EcommerceOrder) error {
	start := time.Now()
	orderCtx := es.createOrderContext(order)

	event.Log(orderCtx, "order processing initiated",
		event.String("order_id", order.ID),
		event.String("customer_id", order.CustomerID),
		event.Int64("item_count", int64(len(order.Items))),
		event.Float64("order_value", order.TotalAmount),
		event.String("business_process", "order_processing"),
	)

	// Step 1: Validate order with detailed telemetry
	if err := es.validateOrder(orderCtx, order); err != nil {
		es.recordBusinessError(orderCtx, "validation_failed", order, err)
		return fmt.Errorf("order validation failed: %w", err)
	}

	// Step 2: Check customer eligibility and tier
	customer, err := es.getCustomerInfo(orderCtx, order.CustomerID)
	if err != nil {
		es.recordBusinessError(orderCtx, "customer_lookup_failed", order, err)
		return fmt.Errorf("customer lookup failed: %w", err)
	}

	// Step 3: Check inventory with retry logic
	if err := es.checkInventoryWithRetry(orderCtx, order); err != nil {
		es.recordBusinessError(orderCtx, "inventory_unavailable", order, err)
		return fmt.Errorf("inventory check failed: %w", err)
	}

	// Step 4: Apply business rules (discounts, limits, etc.)
	if err := es.applyBusinessRules(orderCtx, order, customer); err != nil {
		es.recordBusinessError(orderCtx, "business_rules_violation", order, err)
		return fmt.Errorf("business rules check failed: %w", err)
	}

	// Step 5: Process payment with fraud detection
	paymentResult, err := es.processPaymentSecurely(orderCtx, order, customer)
	if err != nil {
		es.recordBusinessError(orderCtx, "payment_failed", order, err)
		return fmt.Errorf("payment processing failed: %w", err)
	}

	// Step 6: Reserve inventory and fulfill order
	if err := es.fulfillOrder(orderCtx, order); err != nil {
		// Critical path: payment succeeded but fulfillment failed
		event.Log(orderCtx, "critical business error: payment succeeded but fulfillment failed",
			event.String("order_id", order.ID),
			event.String("payment_id", paymentResult.TransactionID),
			event.String("error", err.Error()),
			event.String("severity", "critical"),
			event.String("requires_manual_intervention", "true"),
		)
		es.recordBusinessError(orderCtx, "fulfillment_failed_after_payment", order, err)
		return fmt.Errorf("order fulfillment failed: %w", err)
	}

	// Step 7: Update order status and record success metrics
	order.Status = "confirmed"
	order.UpdatedAt = time.Now()

	duration := time.Since(start)
	es.recordSuccessfulOrder(orderCtx, order, customer, duration)

	event.Log(orderCtx, "order processing completed successfully",
		event.String("order_id", order.ID),
		event.String("customer_id", order.CustomerID),
		event.String("customer_tier", customer.Tier),
		event.String("payment_id", paymentResult.TransactionID),
		event.String("final_status", order.Status),
		event.Int64("total_processing_time_ms", duration.Milliseconds()),
		event.Float64("order_value", order.TotalAmount),
		event.String("business_outcome", "success"),
	)

	return nil
}

func (es *EcommerceService) createOrderContext(order *EcommerceOrder) context.Context {
	// Create enriched context for this order processing session
	orderCtx := es.ctx
	orderCtx = event.WithExporter(orderCtx, event.NewExporter(es.telemetry, eventtest.ExporterOptions()))
	return orderCtx
}

func (es *EcommerceService) validateOrder(ctx context.Context, order *EcommerceOrder) error {
	start := time.Now()

	event.Log(ctx, "validating order",
		event.String("order_id", order.ID),
		event.String("validation_stage", "input_validation"),
	)

	validations := []struct {
		name     string
		validate func() error
	}{
		{"order_id", func() error {
			if order.ID == "" {
				return errors.New("order ID is required")
			}
			return nil
		}},
		{"customer_id", func() error {
			if order.CustomerID == "" {
				return errors.New("customer ID is required")
			}
			return nil
		}},
		{"items", func() error {
			if len(order.Items) == 0 {
				return errors.New("order must contain at least one item")
			}
			return nil
		}},
		{"item_details", func() error {
			for i, item := range order.Items {
				if item.ProductID == "" {
					return fmt.Errorf("item %d: product ID is required", i)
				}
				if item.Quantity <= 0 {
					return fmt.Errorf("item %d: quantity must be positive", i)
				}
				if item.Price < 0 {
					return fmt.Errorf("item %d: price cannot be negative", i)
				}
			}
			return nil
		}},
		{"total_calculation", func() error {
			calculated := 0.0
			for _, item := range order.Items {
				calculated += item.Price * float64(item.Quantity)
			}
			if absFloat(calculated-order.TotalAmount) > 0.01 {
				return fmt.Errorf("total mismatch: calculated %.2f, provided %.2f", calculated, order.TotalAmount)
			}
			return nil
		}},
	}

	for _, validation := range validations {
		if err := validation.validate(); err != nil {
			event.Log(ctx, "validation failed",
				event.String("order_id", order.ID),
				event.String("validation_type", validation.name),
				event.String("error", err.Error()),
			)
			return err
		}

		event.Log(ctx, "validation passed",
			event.String("order_id", order.ID),
			event.String("validation_type", validation.name),
		)
	}

	event.Log(ctx, "order validation completed",
		event.String("order_id", order.ID),
		event.Int64("validation_duration_ms", time.Since(start).Milliseconds()),
		event.String("result", "success"),
	)

	return nil
}

func (es *EcommerceService) getCustomerInfo(ctx context.Context, customerID string) (*Customer, error) {
	start := time.Now()

	event.Log(ctx, "fetching customer information",
		event.String("customer_id", customerID),
		event.String("operation", "customer_lookup"),
	)

	// Simulate database/API call
	time.Sleep(time.Duration(10+rand.Intn(30)) * time.Millisecond)

	// Mock customer data based on ID pattern
	customer := &Customer{
		ID:    customerID,
		Email: fmt.Sprintf("%s@example.com", customerID),
	}

	if strings.Contains(customerID, "premium") {
		customer.Name = "Premium Customer"
		customer.Tier = "premium"
	} else if strings.Contains(customerID, "enterprise") {
		customer.Name = "Enterprise Customer"
		customer.Tier = "enterprise"
	} else {
		customer.Name = "Standard Customer"
		customer.Tier = "standard"
	}

	// Simulate occasional customer lookup failures
	if rand.Float64() < 0.02 {
		return nil, errors.New("customer not found in database")
	}

	duration := time.Since(start)
	event.Log(ctx, "customer information retrieved",
		event.String("customer_id", customerID),
		event.String("customer_tier", customer.Tier),
		event.Int64("lookup_duration_ms", duration.Milliseconds()),
	)

	return customer, nil
}

func (es *EcommerceService) checkInventoryWithRetry(ctx context.Context, order *EcommerceOrder) error {
	maxRetries := 3
	baseDelay := 50 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := es.checkInventory(ctx, order, attempt)
		if err == nil {
			return nil
		}

		event.Log(ctx, "inventory check failed, retrying",
			event.String("order_id", order.ID),
			event.Int64("attempt", int64(attempt)),
			event.Int64("max_retries", int64(maxRetries)),
			event.String("error", err.Error()),
		)

		if attempt < maxRetries {
			// Exponential backoff
			delay := baseDelay * time.Duration(1<<(attempt-1))
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("inventory check failed after %d attempts", maxRetries)
}

func (es *EcommerceService) checkInventory(ctx context.Context, order *EcommerceOrder, attempt int) error {
	start := time.Now()

	event.Log(ctx, "checking inventory availability",
		event.String("order_id", order.ID),
		event.Int64("items_to_check", int64(len(order.Items))),
		event.Int64("attempt", int64(attempt)),
	)

	for _, item := range order.Items {
		itemStart := time.Now()

		// Simulate network call to inventory service
		time.Sleep(time.Duration(15+rand.Intn(35)) * time.Millisecond)

		es.inventoryChecks.Record(ctx, 1,
			event.String("product_id", item.ProductID),
			event.Int64("attempt", int64(attempt)),
		)

		// Simulate inventory availability (95% success rate on first attempt, improving with retries)
		successRate := 0.95 + float64(attempt-1)*0.02
		if rand.Float64() > successRate {
			available := rand.Intn(item.Quantity)
			event.Log(ctx, "insufficient inventory detected",
				event.String("order_id", order.ID),
				event.String("product_id", item.ProductID),
				event.Int64("requested", int64(item.Quantity)),
				event.Int64("available", int64(available)),
				event.Int64("attempt", int64(attempt)),
			)
			return fmt.Errorf("insufficient inventory for product %s: requested %d, available %d",
				item.ProductID, item.Quantity, available)
		}

		event.Log(ctx, "inventory check passed for item",
			event.String("order_id", order.ID),
			event.String("product_id", item.ProductID),
			event.String("product_name", item.Name),
			event.Int64("quantity_requested", int64(item.Quantity)),
			event.Int64("check_duration_ms", time.Since(itemStart).Milliseconds()),
		)
	}

	duration := time.Since(start)
	event.Log(ctx, "inventory check completed successfully",
		event.String("order_id", order.ID),
		event.Int64("total_items", int64(len(order.Items))),
		event.Int64("total_duration_ms", duration.Milliseconds()),
		event.Int64("attempt", int64(attempt)),
	)

	return nil
}

func (es *EcommerceService) applyBusinessRules(ctx context.Context, order *EcommerceOrder, customer *Customer) error {
	event.Log(ctx, "applying business rules",
		event.String("order_id", order.ID),
		event.String("customer_tier", customer.Tier),
		event.Float64("order_value", order.TotalAmount),
	)

	// Rule 1: Order value limits by customer tier
	limits := map[string]float64{
		"standard":   1000.0,
		"premium":    5000.0,
		"enterprise": 50000.0,
	}

	if limit, exists := limits[customer.Tier]; exists && order.TotalAmount > limit {
		return fmt.Errorf("order value %.2f exceeds limit %.2f for %s tier",
			order.TotalAmount, limit, customer.Tier)
	}

	// Rule 2: Item quantity limits
	for _, item := range order.Items {
		maxQty := 10
		if customer.Tier == "enterprise" {
			maxQty = 100
		}
		if item.Quantity > maxQty {
			return fmt.Errorf("quantity %d exceeds maximum %d for product %s",
				item.Quantity, maxQty, item.ProductID)
		}
	}

	// Rule 3: Apply discounts based on customer tier
	originalTotal := order.TotalAmount
	if customer.Tier == "premium" && order.TotalAmount > 500 {
		order.TotalAmount *= 0.95 // 5% discount
	} else if customer.Tier == "enterprise" {
		order.TotalAmount *= 0.90 // 10% discount
	}

	if order.TotalAmount != originalTotal {
		event.Log(ctx, "discount applied",
			event.String("order_id", order.ID),
			event.String("customer_tier", customer.Tier),
			event.Float64("original_total", originalTotal),
			event.Float64("discounted_total", order.TotalAmount),
			event.Float64("discount_amount", originalTotal-order.TotalAmount),
		)
	}

	event.Log(ctx, "business rules applied successfully",
		event.String("order_id", order.ID),
		event.String("customer_tier", customer.Tier),
		event.Float64("final_order_value", order.TotalAmount),
	)

	return nil
}

func (es *EcommerceService) processPaymentSecurely(ctx context.Context, order *EcommerceOrder, customer *Customer) (*EcommercePaymentResult, error) {
	start := time.Now()

	event.Log(ctx, "initiating secure payment processing",
		event.String("order_id", order.ID),
		event.String("customer_id", customer.ID),
		event.String("customer_tier", customer.Tier),
		event.Float64("amount", order.TotalAmount),
		event.String("payment_method", "credit_card"), // Mock
	)

	es.paymentAttempts.Record(ctx, 1,
		event.String("customer_tier", customer.Tier),
		event.Float64("amount", order.TotalAmount),
	)

	// Simulate fraud detection
	time.Sleep(time.Duration(30+rand.Intn(50)) * time.Millisecond)

	// Simulate payment processing
	time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)

	// Simulate different failure scenarios based on amount and customer
	if order.TotalAmount > 10000 && rand.Float64() < 0.05 {
		event.Log(ctx, "payment flagged for manual review",
			event.String("order_id", order.ID),
			event.Float64("amount", order.TotalAmount),
			event.String("reason", "high_value_transaction"),
		)
		return nil, errors.New("payment requires manual review for high-value transaction")
	}

	if rand.Float64() < 0.03 {
		event.Log(ctx, "payment declined by processor",
			event.String("order_id", order.ID),
			event.String("reason", "insufficient_funds"),
		)
		return nil, errors.New("payment declined: insufficient funds")
	}

	if rand.Float64() < 0.01 {
		event.Log(ctx, "payment gateway timeout",
			event.String("order_id", order.ID),
			event.String("error_type", "gateway_timeout"),
		)
		return nil, errors.New("payment gateway timeout - please retry")
	}

	// Generate transaction ID
	transactionID := fmt.Sprintf("txn_%d_%s", time.Now().Unix(), order.ID[:8])

	duration := time.Since(start)
	paymentResult := &EcommercePaymentResult{
		Success:       true,
		TransactionID: transactionID,
		ProcessedAt:   time.Now(),
	}

	event.Log(ctx, "payment processed successfully",
		event.String("order_id", order.ID),
		event.String("customer_id", customer.ID),
		event.String("transaction_id", transactionID),
		event.Float64("amount", order.TotalAmount),
		event.String("customer_tier", customer.Tier),
		event.Int64("processing_duration_ms", duration.Milliseconds()),
		event.String("payment_status", "completed"),
	)

	return paymentResult, nil
}

func (es *EcommerceService) fulfillOrder(ctx context.Context, order *EcommerceOrder) error {
	start := time.Now()

	event.Log(ctx, "initiating order fulfillment",
		event.String("order_id", order.ID),
		event.Int64("items_to_fulfill", int64(len(order.Items))),
	)

	// Simulate inventory reservation
	for _, item := range order.Items {
		time.Sleep(time.Duration(10+rand.Intn(20)) * time.Millisecond)

		event.Log(ctx, "reserving inventory for item",
			event.String("order_id", order.ID),
			event.String("product_id", item.ProductID),
			event.Int64("quantity", int64(item.Quantity)),
		)
	}

	// Simulate fulfillment center assignment
	time.Sleep(time.Duration(20+rand.Intn(40)) * time.Millisecond)

	// Simulate occasional fulfillment failures
	if rand.Float64() < 0.008 {
		event.Log(ctx, "fulfillment center unavailable",
			event.String("order_id", order.ID),
			event.String("reason", "system_maintenance"),
		)
		return errors.New("fulfillment center temporarily unavailable")
	}

	duration := time.Since(start)
	event.Log(ctx, "order fulfillment completed",
		event.String("order_id", order.ID),
		event.Int64("items_fulfilled", int64(len(order.Items))),
		event.Int64("fulfillment_duration_ms", duration.Milliseconds()),
		event.String("fulfillment_status", "ready_for_shipping"),
	)

	return nil
}

func (es *EcommerceService) recordSuccessfulOrder(ctx context.Context, order *EcommerceOrder, customer *Customer, duration time.Duration) {
	// Record business metrics
	es.ordersProcessed.Record(ctx, 1,
		event.String("customer_tier", customer.Tier),
		event.String("order_status", "success"),
	)

	es.orderValue.Record(ctx, int64(order.TotalAmount),
		event.String("customer_tier", customer.Tier),
		event.String("order_status", "success"),
	)

	es.processingTime.Record(ctx, duration,
		event.String("customer_tier", customer.Tier),
		event.Int64("item_count", int64(len(order.Items))),
	)

	// Simulate customer satisfaction (higher for premium customers)
	satisfactionScore := 4.0 + rand.Float64()
	if customer.Tier == "premium" {
		satisfactionScore = 4.2 + rand.Float64()*0.8
	} else if customer.Tier == "enterprise" {
		satisfactionScore = 4.5 + rand.Float64()*0.5
	}

	es.customerSatisfaction.Record(ctx, satisfactionScore,
		event.String("customer_tier", customer.Tier),
		event.String("order_id", order.ID),
	)
}

func (es *EcommerceService) recordBusinessError(ctx context.Context, errorType string, order *EcommerceOrder, err error) {
	es.errorRate.Record(ctx, 1,
		event.String("error_type", errorType),
		event.String("order_value_range", getOrderValueRange(order.TotalAmount)),
	)

	event.Log(ctx, "business process error occurred",
		event.String("order_id", order.ID),
		event.String("customer_id", order.CustomerID),
		event.String("error_type", errorType),
		event.String("error_message", err.Error()),
		event.Float64("order_value", order.TotalAmount),
		event.Int64("item_count", int64(len(order.Items))),
		event.String("business_impact", "order_failed"),
	)
}

// Helper functions
func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func getOrderValueRange(amount float64) string {
	switch {
	case amount < 50:
		return "low"
	case amount < 200:
		return "medium"
	case amount < 1000:
		return "high"
	default:
		return "very_high"
	}
}

// HTTP API handlers for the e-commerce service
func (es *EcommerceService) setupHTTPHandlers() *http.ServeMux {
	mux := http.NewServeMux()

	// Business API endpoints
	mux.HandleFunc("/api/orders", es.handleOrders)
	mux.HandleFunc("/api/customers/", es.handleCustomers)
	mux.HandleFunc("/api/inventory/", es.handleInventory)

	// Operational endpoints
	mux.HandleFunc("/health", es.handleHealth)
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.HandlerFor(es.prometheusReg, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	})

	return mux
}

func (es *EcommerceService) handleOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = event.WithExporter(ctx, event.NewExporter(es.telemetry, eventtest.ExporterOptions()))

	if r.Method == "POST" {
		es.handleCreateOrder(ctx, w, r)
	} else if r.Method == "GET" {
		es.handleListOrders(ctx, w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (es *EcommerceService) handleCreateOrder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	event.Log(ctx, "order creation API called",
		event.String("endpoint", "/api/orders"),
		event.String("method", "POST"),
		event.String("user_agent", r.UserAgent()),
	)

	var order EcommerceOrder
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		event.Log(ctx, "invalid order JSON",
			event.String("error", err.Error()),
		)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set creation timestamp
	order.CreatedAt = time.Now()
	order.Status = "processing"

	// Process the order
	if err := es.ProcessOrder(&order); err != nil {
		event.Log(ctx, "order processing failed via API",
			event.String("order_id", order.ID),
			event.String("error", err.Error()),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":    "Order processing failed",
			"details":  err.Error(),
			"order_id": order.ID,
		})
		return
	}

	// Return success response
	duration := time.Since(start)
	event.Log(ctx, "order created successfully via API",
		event.String("order_id", order.ID),
		event.String("status", order.Status),
		event.Int64("api_duration_ms", duration.Milliseconds()),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (es *EcommerceService) handleListOrders(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	event.Log(ctx, "order list API called",
		event.String("endpoint", "/api/orders"),
		event.String("method", "GET"),
	)

	// Mock order list (in real app, would fetch from database)
	orders := []EcommerceOrder{
		{
			ID:          "order_001",
			CustomerID:  "customer_premium_001",
			Status:      "confirmed",
			TotalAmount: 299.99,
			CreatedAt:   time.Now().Add(-1 * time.Hour),
		},
		{
			ID:          "order_002",
			CustomerID:  "customer_standard_002",
			Status:      "processing",
			TotalAmount: 89.99,
			CreatedAt:   time.Now().Add(-30 * time.Minute),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orders": orders,
		"total":  len(orders),
	})
}

func (es *EcommerceService) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = event.WithExporter(ctx, event.NewExporter(es.telemetry, eventtest.ExporterOptions()))

	event.Log(ctx, "health check requested")

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "ecommerce",
		"version":   "1.0.0",
		"components": map[string]string{
			"database":    "healthy",
			"payment_api": "healthy",
			"inventory":   "healthy",
			"fulfillment": "healthy",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (es *EcommerceService) handleCustomers(w http.ResponseWriter, r *http.Request) {
	// Extract customer ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/customers/")
	customerID := strings.Split(path, "/")[0]

	ctx := r.Context()
	ctx = event.WithExporter(ctx, event.NewExporter(es.telemetry, eventtest.ExporterOptions()))

	event.Log(ctx, "customer API called",
		event.String("customer_id", customerID),
		event.String("method", r.Method),
	)

	customer, err := es.getCustomerInfo(ctx, customerID)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customer)
}

func (es *EcommerceService) handleInventory(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/inventory/")
	productID := strings.Split(path, "/")[0]

	ctx := r.Context()
	ctx = event.WithExporter(ctx, event.NewExporter(es.telemetry, eventtest.ExporterOptions()))

	event.Log(ctx, "inventory API called",
		event.String("product_id", productID),
	)

	// Mock inventory data
	inventory := &InventoryItem{
		ProductID: productID,
		Available: 100 + rand.Intn(900),
		Reserved:  rand.Intn(50),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inventory)
}

// Demo function to run the e-commerce service
func runEcommerceDemo() {
	fmt.Println("🛒 E-commerce Service with Advanced Telemetry")
	fmt.Println("=============================================")

	service, err := NewEcommerceService()
	if err != nil {
		fmt.Printf("❌ Failed to create service: %v\n", err)
		return
	}

	// Demo 1: Process some sample orders
	fmt.Println("\n📦 Processing sample orders...")
	sampleOrders := []*EcommerceOrder{
		{
			ID:         "order_demo_001",
			CustomerID: "customer_premium_alice",
			Items: []EcommerceItem{
				{ProductID: "laptop_pro_001", Name: "MacBook Pro", Quantity: 1, Price: 2499.99},
				{ProductID: "mouse_wireless_001", Name: "Wireless Mouse", Quantity: 1, Price: 79.99},
			},
			TotalAmount: 2579.98,
		},
		{
			ID:         "order_demo_002",
			CustomerID: "customer_standard_bob",
			Items: []EcommerceItem{
				{ProductID: "book_programming_001", Name: "Go Programming", Quantity: 2, Price: 45.99},
				{ProductID: "coffee_mug_001", Name: "Developer Mug", Quantity: 1, Price: 15.99},
			},
			TotalAmount: 107.97,
		},
		{
			ID:         "order_demo_003",
			CustomerID: "customer_enterprise_acme",
			Items: []EcommerceItem{
				{ProductID: "server_rack_001", Name: "Server Rack", Quantity: 5, Price: 1299.99},
			},
			TotalAmount: 6499.95,
		},
	}

	successCount := 0
	for i, order := range sampleOrders {
		fmt.Printf("\n--- Processing Order %d: %s ---\n", i+1, order.ID)

		if err := service.ProcessOrder(order); err != nil {
			fmt.Printf("❌ Order %s failed: %v\n", order.ID, err)
		} else {
			fmt.Printf("✅ Order %s processed successfully (Status: %s)\n", order.ID, order.Status)
			successCount++
		}

		time.Sleep(500 * time.Millisecond) // Simulate time between orders
	}

	fmt.Printf("\n📊 Processing Summary: %d/%d orders successful\n", successCount, len(sampleOrders))

	// Demo 2: Start HTTP server for API testing
	fmt.Println("\n🌐 Starting HTTP API server...")
	mux := service.setupHTTPHandlers()

	fmt.Println("API Endpoints available:")
	fmt.Println("  📝 POST /api/orders          - Create new order")
	fmt.Println("  📋 GET  /api/orders          - List orders")
	fmt.Println("  👤 GET  /api/customers/{id}  - Get customer info")
	fmt.Println("  📦 GET  /api/inventory/{id}  - Check inventory")
	fmt.Println("  ❤️  GET  /health             - Health check")
	fmt.Println("  📊 GET  /metrics             - Prometheus metrics")

	fmt.Println("\n🚀 Server starting on http://localhost:8081")
	fmt.Println("💡 Try these commands:")
	fmt.Println(`  curl -X POST http://localhost:8081/api/orders \`)
	fmt.Println(`    -H "Content-Type: application/json" \`)
	fmt.Println(`    -d '{"id":"test_001","customer_id":"customer_premium_test","items":[{"product_id":"test_product","name":"Test Item","quantity":1,"price":99.99}],"total_amount":99.99}'`)
	fmt.Println(`  curl http://localhost:8081/metrics`)
	fmt.Println(`  curl http://localhost:8081/health`)

	if err := http.ListenAndServe(":8081", mux); err != nil {
		fmt.Printf("❌ Server failed: %v\n", err)
	}
}

// Uncomment to run as standalone application
/*
func main() {
	runEcommerceDemo()
}
*/
