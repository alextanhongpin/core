package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/alextanhongpin/core/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/slog"
)

// EcommerceOrderService demonstrates advanced telemetry in a business domain
type EcommerceOrderService struct {
	telemetry *telemetry.MultiHandler
	ctx       context.Context

	// Business metrics
	ordersProcessed *event.Counter
	orderValue      *event.Counter
	processingTime  *event.Duration
	inventoryChecks *event.Counter
	paymentAttempts *event.Counter
	errorRate       *event.Counter
}

type Order struct {
	ID          string    `json:"id"`
	CustomerID  string    `json:"customer_id"`
	Items       []Item    `json:"items"`
	TotalAmount float64   `json:"total_amount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Item struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type PaymentResult struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transaction_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

func NewEcommerceOrderService() (*EcommerceOrderService, error) {
	// Setup structured logging with rich context
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Add custom formatting for business context
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339))
			}
			if a.Key == slog.SourceKey {
				return slog.String("source", a.Value.String())
			}
			return a
		},
	}))

	slogHandler, err := telemetry.NewSlogHandler(logger, telemetry.WithSlogErrorHandler(func(err error) {
		logger.Error("telemetry error", "error", err)
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create slog handler: %w", err)
	}

	// Setup Prometheus with custom error handling
	prometheusReg := prometheus.NewRegistry()
	prometheusHandler, err := telemetry.NewPrometheusHandler(prometheusReg, telemetry.WithPrometheusErrorHandler(func(err error) {
		logger.Error("prometheus error", "error", err)
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus handler: %w", err)
	}

	// Create multi-handler for unified telemetry
	multiHandler := &telemetry.MultiHandler{
		Log:    slogHandler,
		Metric: prometheusHandler,
	}

	// Setup event context
	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(multiHandler, eventtest.ExporterOptions()))

	service := &EcommerceOrderService{
		telemetry: multiHandler,
		ctx:       ctx,
	}

	service.setupMetrics()
	return service, nil
}

func (os *EcommerceOrderService) setupMetrics() {
	// Define business-specific metrics
	os.ordersProcessed = event.NewCounter("orders_processed_total", &event.MetricOptions{
		Description: "Total number of orders processed",
	})

	os.orderValue = event.NewCounter("order_value_total", &event.MetricOptions{
		Description: "Total monetary value of orders processed",
		Unit:        "dollars",
	})

	os.processingTime = event.NewDuration("order_processing_duration", &event.MetricOptions{
		Description: "Time taken to process an order",
	})

	os.inventoryChecks = event.NewCounter("inventory_checks_total", &event.MetricOptions{
		Description: "Number of inventory checks performed",
	})

	os.paymentAttempts = event.NewCounter("payment_attempts_total", &event.MetricOptions{
		Description: "Number of payment processing attempts",
	})

	os.errorRate = event.NewCounter("order_errors_total", &event.MetricOptions{
		Description: "Number of order processing errors",
	})
}

// ProcessOrder demonstrates comprehensive telemetry throughout a business process
func (os *EcommerceOrderService) ProcessOrder(order *Order) error {
	start := time.Now()

	// Create order-scoped context with telemetry
	ctx := event.NewContext(os.ctx, os.telemetry)

	// Enrich context with order information
	event.Log(ctx, "order processing started",
		event.String("order_id", order.ID),
		event.String("customer_id", order.CustomerID),
		event.Int64("item_count", int64(len(order.Items))),
		event.Float64("total_amount", order.TotalAmount),
	)

	// Step 1: Validate order
	if err := os.validateOrder(ctx, order); err != nil {
		os.recordError(ctx, "validation", order, err)
		return fmt.Errorf("order validation failed: %w", err)
	}

	// Step 2: Check inventory for all items
	if err := os.checkInventory(ctx, order); err != nil {
		os.recordError(ctx, "inventory", order, err)
		return fmt.Errorf("inventory check failed: %w", err)
	}

	// Step 3: Process payment
	paymentResult, err := os.processPayment(ctx, order)
	if err != nil {
		os.recordError(ctx, "payment", order, err)
		return fmt.Errorf("payment processing failed: %w", err)
	}

	// Step 4: Reserve inventory
	if err := os.reserveInventory(ctx, order); err != nil {
		// This is a critical error - payment succeeded but inventory failed
		event.Log(ctx, "critical error: payment succeeded but inventory reservation failed",
			event.String("order_id", order.ID),
			event.String("payment_id", paymentResult.TransactionID),
			event.String("error", err.Error()),
		)
		os.recordError(ctx, "inventory_reservation", order, err)
		return fmt.Errorf("inventory reservation failed: %w", err)
	}

	// Step 5: Update order status
	order.Status = "confirmed"

	// Record successful metrics
	duration := time.Since(start)
	os.ordersProcessed.Record(ctx, 1,
		event.String("status", "success"),
		event.String("customer_type", os.getCustomerType(order.CustomerID)),
	)

	os.orderValue.Record(ctx, order.TotalAmount,
		event.String("status", "success"),
		event.String("customer_type", os.getCustomerType(order.CustomerID)),
	)

	os.processingTime.Record(ctx, duration,
		event.String("status", "success"),
		event.Int64("item_count", int64(len(order.Items))),
	)

	event.Log(ctx, "order processing completed successfully",
		event.String("order_id", order.ID),
		event.String("customer_id", order.CustomerID),
		event.String("payment_id", paymentResult.TransactionID),
		event.String("status", order.Status),
		event.Int64("processing_time_ms", duration.Milliseconds()),
		event.Float64("order_value", order.TotalAmount),
	)

	return nil
}

func (os *EcommerceOrderService) validateOrder(ctx context.Context, order *Order) error {
	event.Log(ctx, "validating order",
		event.String("order_id", order.ID),
		event.Int64("item_count", int64(len(order.Items))),
	)

	if order.ID == "" {
		return errors.New("order ID is required")
	}

	if order.CustomerID == "" {
		return errors.New("customer ID is required")
	}

	if len(order.Items) == 0 {
		return errors.New("order must contain at least one item")
	}

	var calculatedTotal float64
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
		calculatedTotal += item.Price * float64(item.Quantity)
	}

	// Check if calculated total matches order total (with tolerance for floating point)
	if abs(calculatedTotal-order.TotalAmount) > 0.01 {
		return fmt.Errorf("order total mismatch: calculated %.2f, provided %.2f", calculatedTotal, order.TotalAmount)
	}

	event.Log(ctx, "order validation passed",
		event.String("order_id", order.ID),
		event.Float64("validated_total", calculatedTotal),
	)

	return nil
}

func (os *EcommerceOrderService) checkInventory(ctx context.Context, order *Order) error {
	start := time.Now()

	event.Log(ctx, "checking inventory for order",
		event.String("order_id", order.ID),
		event.Int64("items_to_check", int64(len(order.Items))),
	)

	// Simulate inventory checks for each item
	for _, item := range order.Items {
		itemStart := time.Now()

		// Simulate network call to inventory service
		time.Sleep(time.Duration(10+rand.Intn(40)) * time.Millisecond)

		// Simulate occasional inventory shortage (5% chance)
		if rand.Float64() < 0.05 {
			event.Log(ctx, "insufficient inventory",
				event.String("order_id", order.ID),
				event.String("product_id", item.ProductID),
				event.Int64("requested", int64(item.Quantity)),
				event.Int64("available", int64(rand.Intn(item.Quantity))),
			)
			return fmt.Errorf("insufficient inventory for product %s", item.ProductID)
		}

		os.inventoryChecks.Record(ctx, 1,
			event.String("product_id", item.ProductID),
			event.String("status", "success"),
		)

		event.Log(ctx, "inventory check passed",
			event.String("order_id", order.ID),
			event.String("product_id", item.ProductID),
			event.Int64("quantity", int64(item.Quantity)),
			event.Int64("check_duration_ms", time.Since(itemStart).Milliseconds()),
		)
	}

	inventoryLatency := event.NewDuration("inventory_check_duration", &event.MetricOptions{
		Description: "Time taken for inventory checks",
	})

	duration := time.Since(start)
	inventoryLatency.Record(ctx, duration,
		event.String("order_id", order.ID),
		event.Int64("items_checked", int64(len(order.Items))),
	)

	event.Log(ctx, "inventory check completed",
		event.String("order_id", order.ID),
		event.Int64("total_items_checked", int64(len(order.Items))),
		event.Int64("total_duration_ms", duration.Milliseconds()),
	)

	return nil
}

func (os *EcommerceOrderService) processPayment(ctx context.Context, order *Order) (*PaymentResult, error) {
	start := time.Now()

	event.Log(ctx, "processing payment",
		event.String("order_id", order.ID),
		event.String("customer_id", order.CustomerID),
		event.Float64("amount", order.TotalAmount),
	)

	os.paymentAttempts.Record(ctx, 1,
		event.String("customer_id", order.CustomerID),
		event.Float64("amount", order.TotalAmount),
	)

	// Simulate payment processing time
	time.Sleep(time.Duration(50+rand.Intn(150)) * time.Millisecond)

	// Simulate payment failures (3% chance)
	if rand.Float64() < 0.03 {
		event.Log(ctx, "payment failed",
			event.String("order_id", order.ID),
			event.String("customer_id", order.CustomerID),
			event.Float64("amount", order.TotalAmount),
			event.String("reason", "insufficient_funds"),
		)
		return nil, errors.New("payment declined: insufficient funds")
	}

	// Simulate payment processing errors (1% chance)
	if rand.Float64() < 0.01 {
		event.Log(ctx, "payment processing error",
			event.String("order_id", order.ID),
			event.String("customer_id", order.CustomerID),
			event.Float64("amount", order.TotalAmount),
			event.String("reason", "gateway_timeout"),
		)
		return nil, errors.New("payment gateway timeout")
	}

	// Generate transaction ID
	transactionID := fmt.Sprintf("txn_%d_%s", time.Now().Unix(), order.ID[:8])

	paymentAmountHistogram := event.NewDuration("payment_amount", &event.MetricOptions{
		Description: "Payment amounts processed",
		Unit:        "dollars",
	})

	duration := time.Since(start)
	paymentAmountHistogram.Record(ctx, duration,
		event.String("customer_type", os.getCustomerType(order.CustomerID)),
		event.String("status", "success"),
	)

	event.Log(ctx, "payment processed successfully",
		event.String("order_id", order.ID),
		event.String("customer_id", order.CustomerID),
		event.String("transaction_id", transactionID),
		event.Float64("amount", order.TotalAmount),
		event.Int64("processing_time_ms", duration.Milliseconds()),
	)

	return &PaymentResult{
		Success:       true,
		TransactionID: transactionID,
	}, nil
}

func (os *EcommerceOrderService) reserveInventory(ctx context.Context, order *Order) error {
	event.Log(ctx, "reserving inventory",
		event.String("order_id", order.ID),
		event.Int64("items_to_reserve", int64(len(order.Items))),
	)

	// Simulate inventory reservation
	time.Sleep(time.Duration(20+rand.Intn(30)) * time.Millisecond)

	// Simulate rare reservation failures (1% chance)
	if rand.Float64() < 0.01 {
		event.Log(ctx, "inventory reservation failed",
			event.String("order_id", order.ID),
			event.String("reason", "concurrent_modification"),
		)
		return errors.New("inventory reservation failed due to concurrent modification")
	}

	event.Log(ctx, "inventory reserved successfully",
		event.String("order_id", order.ID),
		event.Int64("items_reserved", int64(len(order.Items))),
	)

	return nil
}

func (os *EcommerceOrderService) recordError(ctx context.Context, stage string, order *Order, err error) {
	os.errorRate.Record(ctx, 1,
		event.String("stage", stage),
		event.String("error_type", getErrorType(err)),
		event.String("customer_type", os.getCustomerType(order.CustomerID)),
	)

	event.Log(ctx, "order processing error",
		event.String("order_id", order.ID),
		event.String("customer_id", order.CustomerID),
		event.String("stage", stage),
		event.String("error", err.Error()),
		event.Float64("order_value", order.TotalAmount),
	)
}

func (os *EcommerceOrderService) getCustomerType(customerID string) string {
	// Simple logic to categorize customers
	if len(customerID) > 10 {
		return "premium"
	}
	return "standard"
}

func getErrorType(err error) string {
	errStr := err.Error()
	switch {
	case contains(errStr, "validation"):
		return "validation_error"
	case contains(errStr, "inventory"):
		return "inventory_error"
	case contains(errStr, "payment"):
		return "payment_error"
	case contains(errStr, "timeout"):
		return "timeout_error"
	default:
		return "unknown_error"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// runEcommerceExample demonstrates the e-commerce order service
func runEcommerceExample() {
	service, err := NewEcommerceOrderService()
	if err != nil {
		fmt.Printf("Failed to create service: %v\n", err)
		return
	}

	fmt.Println("=== E-commerce Order Processing Demo ===")
	fmt.Println("Processing sample orders with comprehensive telemetry...")

	// Create sample orders
	orders := []*Order{
		{
			ID:         "order_001",
			CustomerID: "customer_premium_001",
			Items: []Item{
				{ProductID: "laptop_001", Quantity: 1, Price: 999.99},
				{ProductID: "mouse_001", Quantity: 2, Price: 29.99},
			},
			TotalAmount: 1059.97,
			CreatedAt:   time.Now(),
		},
		{
			ID:         "order_002",
			CustomerID: "customer_std",
			Items: []Item{
				{ProductID: "book_001", Quantity: 3, Price: 15.99},
				{ProductID: "pen_001", Quantity: 5, Price: 2.50},
			},
			TotalAmount: 60.47,
			CreatedAt:   time.Now(),
		},
		{
			ID:         "order_003",
			CustomerID: "customer_premium_002",
			Items: []Item{
				{ProductID: "phone_001", Quantity: 1, Price: 699.99},
			},
			TotalAmount: 699.99,
			CreatedAt:   time.Now(),
		},
	}

	// Process orders
	for i, order := range orders {
		fmt.Printf("\n--- Processing Order %d ---\n", i+1)

		if err := service.ProcessOrder(order); err != nil {
			fmt.Printf("❌ Order %s failed: %v\n", order.ID, err)
		} else {
			fmt.Printf("✅ Order %s processed successfully\n", order.ID)
		}

		// Add some delay between orders
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n=== Demo completed ===")
	fmt.Println("Check the console output above to see structured logs with:")
	fmt.Println("- Order processing stages")
	fmt.Println("- Business metrics")
	fmt.Println("- Error handling and recovery")
	fmt.Println("- Performance monitoring")
}

// func main() {
// 	runEcommerceExample()
// }
