package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/alextanhongpin/core/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	redis "github.com/redis/go-redis/v9"
)

// Simulated database
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Order struct {
	ID     int     `json:"id"`
	UserID int     `json:"user_id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

// In-memory stores for demo
var (
	users       = make(map[int]User)
	orders      = make(map[int]Order)
	nextUserID  = 1
	nextOrderID = 1
)

// Application struct with metrics
type EcommerceAPI struct {
	tracker *metrics.Tracker
	logger  *slog.Logger
	client  *redis.Client
}

func NewEcommerceAPI() *EcommerceAPI {
	// Redis client for advanced analytics
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Redis connection failed (analytics will be disabled): %v", err)
		client = nil
	}

	return &EcommerceAPI{
		tracker: func() *metrics.Tracker {
			if client != nil {
				return metrics.NewTracker("ecommerce_api", client)
			}
			return nil
		}(),
		logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})),
		client: client,
	}
}

func (api *EcommerceAPI) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Register Prometheus metrics
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		metrics.InFlightGauge,
		metrics.RequestDuration,
		metrics.ResponseSize,
		metrics.RED,
	)

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		health := map[string]string{"status": "ok", "timestamp": time.Now().Format(time.RFC3339)}
		if api.client != nil {
			if err := api.client.Ping(r.Context()).Err(); err != nil {
				health["redis"] = "disconnected"
			} else {
				health["redis"] = "connected"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	})

	// Analytics endpoint (if Redis is available)
	if api.tracker != nil {
		mux.Handle("/admin/analytics", metrics.TrackerStatsHandler(api.tracker))
	}

	// API endpoints with full instrumentation
	mux.Handle("/api/users", api.instrumentHandler("list_users", http.HandlerFunc(api.listUsers)))
	mux.Handle("/api/users/create", api.instrumentHandler("create_user", http.HandlerFunc(api.createUser)))
	mux.Handle("/api/orders", api.instrumentHandler("list_orders", http.HandlerFunc(api.listOrders)))
	mux.Handle("/api/orders/create", api.instrumentHandler("create_order", http.HandlerFunc(api.createOrder)))
	mux.Handle("/api/orders/process", api.instrumentHandler("process_order", http.HandlerFunc(api.processOrder)))

	// Simulate load endpoint for testing
	mux.Handle("/api/load-test", api.instrumentHandler("load_test", http.HandlerFunc(api.loadTest)))

	return mux
}

func (api *EcommerceAPI) instrumentHandler(action string, handler http.Handler) http.Handler {
	// Add request duration tracking
	instrumentedHandler := metrics.RequestDurationHandler("v2.1", handler)

	// Add RED metrics tracking
	redHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		red := metrics.NewRED("ecommerce_api", action)
		defer red.Done()

		// Track in-flight requests
		metrics.InFlightGauge.Inc()
		defer metrics.InFlightGauge.Dec()

		// Custom response writer to capture status
		rw := &responseWriter{ResponseWriter: w, statusCode: 200}
		instrumentedHandler.ServeHTTP(rw, r)

		// Set RED status based on HTTP status
		if rw.statusCode >= 400 {
			if rw.statusCode >= 500 {
				red.SetStatus("server_error")
			} else {
				red.SetStatus("client_error")
			}
		}

		// Observe response size
		metrics.ObserveResponseSize(r)
	})

	// Add advanced analytics if available
	if api.tracker != nil {
		userExtractor := func(r *http.Request) string {
			// In a real app, this would extract user ID from JWT or session
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				userID = r.RemoteAddr // fallback to IP
			}
			return userID
		}

		return metrics.TrackerHandler(redHandler, api.tracker, userExtractor, api.logger)
	}

	return redHandler
}

// Custom response writer to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// API Handlers with realistic business logic

func (api *EcommerceAPI) listUsers(w http.ResponseWriter, r *http.Request) {
	// Simulate database query time
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

	userList := make([]User, 0, len(users))
	for _, user := range users {
		userList = append(userList, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": userList,
		"count": len(userList),
	})
}

func (api *EcommerceAPI) createUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Simulate validation time
	time.Sleep(time.Duration(rand.Intn(30)) * time.Millisecond)

	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Email == "" {
		http.Error(w, "Name and email required", http.StatusBadRequest)
		return
	}

	// Simulate database insert time
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

	user := User{
		ID:    nextUserID,
		Name:  req.Name,
		Email: req.Email,
	}
	users[nextUserID] = user
	nextUserID++

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (api *EcommerceAPI) listOrders(w http.ResponseWriter, r *http.Request) {
	// Simulate database query time
	time.Sleep(time.Duration(rand.Intn(75)) * time.Millisecond)

	orderList := make([]Order, 0, len(orders))
	for _, order := range orders {
		orderList = append(orderList, order)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orders": orderList,
		"count":  len(orderList),
	})
}

func (api *EcommerceAPI) createOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID int     `json:"user_id"`
		Amount float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate user exists
	if _, exists := users[req.UserID]; !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Simulate inventory check (occasionally fails)
	time.Sleep(time.Duration(rand.Intn(150)) * time.Millisecond)
	if rand.Intn(10) == 0 { // 10% failure rate
		http.Error(w, "Product out of stock", http.StatusConflict)
		return
	}

	order := Order{
		ID:     nextOrderID,
		UserID: req.UserID,
		Amount: req.Amount,
		Status: "created",
	}
	orders[nextOrderID] = order
	nextOrderID++

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (api *EcommerceAPI) processOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orderIDStr := r.URL.Query().Get("id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	order, exists := orders[orderID]
	if !exists {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Simulate payment processing time
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

	// Simulate payment failures
	if rand.Intn(20) == 0 { // 5% failure rate
		http.Error(w, "Payment processing failed", http.StatusPaymentRequired)
		return
	}

	// Simulate processing errors
	if rand.Intn(50) == 0 { // 2% failure rate
		http.Error(w, "Internal processing error", http.StatusInternalServerError)
		return
	}

	order.Status = "processed"
	orders[orderID] = order

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (api *EcommerceAPI) loadTest(w http.ResponseWriter, r *http.Request) {
	// Simulate variable response times
	sleepTime := time.Duration(rand.Intn(200)) * time.Millisecond
	time.Sleep(sleepTime)

	// Simulate occasional errors
	if rand.Intn(25) == 0 { // 4% error rate
		http.Error(w, "Simulated load test error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":     "Load test endpoint",
		"sleep_time":  sleepTime.String(),
		"timestamp":   time.Now().Format(time.RFC3339),
		"random_data": rand.Intn(1000),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	api := NewEcommerceAPI()
	mux := api.SetupRoutes()

	// Seed some initial data
	seedData()

	port := ":8080"
	fmt.Printf("🚀 E-commerce API running on port %s\n", port)
	fmt.Println("📊 Endpoints:")
	fmt.Println("  - Metrics: http://localhost:8080/metrics")
	fmt.Println("  - Health: http://localhost:8080/health")
	if api.tracker != nil {
		fmt.Println("  - Analytics: http://localhost:8080/admin/analytics")
	}
	fmt.Println("  - API Docs: http://localhost:8080/api/")
	fmt.Println("\n🧪 Test endpoints:")
	fmt.Println("  curl -X POST http://localhost:8080/api/users/create -d '{\"name\":\"John\",\"email\":\"john@example.com\"}'")
	fmt.Println("  curl http://localhost:8080/api/users")
	fmt.Println("  curl -X POST http://localhost:8080/api/orders/create -d '{\"user_id\":1,\"amount\":99.99}'")
	fmt.Println("  curl http://localhost:8080/api/load-test")

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func seedData() {
	// Create some initial users
	users[1] = User{ID: 1, Name: "Alice Johnson", Email: "alice@example.com"}
	users[2] = User{ID: 2, Name: "Bob Smith", Email: "bob@example.com"}
	users[3] = User{ID: 3, Name: "Carol Davis", Email: "carol@example.com"}
	nextUserID = 4

	// Create some initial orders
	orders[1] = Order{ID: 1, UserID: 1, Amount: 29.99, Status: "processed"}
	orders[2] = Order{ID: 2, UserID: 2, Amount: 59.99, Status: "created"}
	nextOrderID = 3

	fmt.Println("✅ Seeded initial data: 3 users, 2 orders")
}
