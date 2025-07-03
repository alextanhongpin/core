package handlers_test

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/alextanhongpin/core/types/handlers"
)

// Example: Basic request/response handling
func ExampleRouter_basic() {
	router := handlers.NewRouter()

	// Register a simple handler
	router.HandleFunc("greet", func(w handlers.ResponseWriter, r *handlers.Request) error {
		type GreetRequest struct {
			Name string `json:"name"`
		}

		var req GreetRequest
		if err := r.Decode(&req); err != nil {
			w.WriteStatus(400)
			return w.Encode(map[string]string{"error": "invalid request"})
		}

		response := map[string]string{
			"message": fmt.Sprintf("Hello, %s!", req.Name),
		}
		return w.Encode(response)
	})

	// Process a request
	req := handlers.NewRequest("greet", strings.NewReader(`{"name": "Alice"}`))
	resp, err := router.Do(req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", resp.Status)
	fmt.Printf("Response: %s", resp.String())

	// Output:
	// Status: 200
	// Response: {"message":"Hello, Alice!"}
}

// Example: Using middleware for logging and recovery
func ExampleRouter_middleware() {
	router := handlers.NewRouter()

	// Add logging middleware
	router.Use(handlers.LoggingMiddleware(func(pattern string, duration time.Duration, status int) {
		fmt.Printf("[LOG] %s - %d - %v\n", pattern, status, duration)
	}))

	// Add recovery middleware
	router.Use(handlers.RecoveryMiddleware())

	// Handler that might panic
	router.HandleFunc("risky", func(w handlers.ResponseWriter, r *handlers.Request) error {
		panic("something went wrong!")
	})

	// Handler that works normally
	router.HandleFunc("safe", func(w handlers.ResponseWriter, r *handlers.Request) error {
		return w.Encode(map[string]string{"status": "ok"})
	})

	// Test the risky handler
	req := handlers.NewRequest("risky", strings.NewReader("{}"))
	resp, _ := router.Do(req)
	fmt.Printf("Risky handler status: %d\n", resp.Status)

	// Test the safe handler
	req = handlers.NewRequest("safe", strings.NewReader("{}"))
	resp, _ = router.Do(req)
	fmt.Printf("Safe handler status: %d\n", resp.Status)

	// Output:
	// [LOG] risky - 500 - [duration]
	// Risky handler status: 500
	// [LOG] safe - 200 - [duration]
	// Safe handler status: 200
}

// Real-world example: User service with authentication
type UserService struct {
	users map[string]User
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func NewUserService() *UserService {
	return &UserService{
		users: make(map[string]User),
	}
}

func (us *UserService) createUserHandler(w handlers.ResponseWriter, r *handlers.Request) error {
	var req CreateUserRequest
	if err := r.Decode(&req); err != nil {
		w.WriteStatus(400)
		return w.Encode(map[string]string{"error": "invalid request body"})
	}

	// Validate request
	if req.Name == "" || req.Email == "" {
		w.WriteStatus(400)
		return w.Encode(map[string]string{"error": "name and email are required"})
	}

	// Create user
	userID := fmt.Sprintf("user_%d", len(us.users)+1)
	user := User{
		ID:    userID,
		Name:  req.Name,
		Email: req.Email,
	}
	us.users[userID] = user

	w.WriteStatus(201)
	w.SetHeader("Location", fmt.Sprintf("/users/%s", userID))
	return w.Encode(user)
}

func (us *UserService) getUserHandler(w handlers.ResponseWriter, r *handlers.Request) error {
	userID, ok := r.GetMeta("user_id")
	if !ok {
		w.WriteStatus(400)
		return w.Encode(map[string]string{"error": "user_id parameter required"})
	}

	user, exists := us.users[userID]
	if !exists {
		w.WriteStatus(404)
		return w.Encode(map[string]string{"error": "user not found"})
	}

	return w.Encode(user)
}

func (us *UserService) listUsersHandler(w handlers.ResponseWriter, r *handlers.Request) error {
	users := make([]User, 0, len(us.users))
	for _, user := range us.users {
		users = append(users, user)
	}

	return w.Encode(map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

func ExampleUserService() {
	service := NewUserService()
	router := handlers.NewRouter()

	// Add authentication middleware
	router.Use(handlers.AuthMiddleware(func(token string) error {
		if token != "Bearer valid-token" {
			return fmt.Errorf("invalid token")
		}
		return nil
	}))

	// Register handlers
	router.HandleFunc("users.create", service.createUserHandler)
	router.HandleFunc("users.get", service.getUserHandler)
	router.HandleFunc("users.list", service.listUsersHandler)

	// Create a user
	createReq := handlers.NewRequest("users.create",
		strings.NewReader(`{"name":"John Doe","email":"john@example.com"}`))
	createReq.WithMeta("Authorization", "Bearer valid-token")

	createResp, err := router.Do(createReq)
	if err != nil {
		log.Printf("Create error: %v", err)
		return
	}

	fmt.Printf("Create User - Status: %d\n", createResp.Status)
	fmt.Printf("Location Header: %s\n", createResp.Headers["Location"])

	// Get the user
	getReq := handlers.NewRequest("users.get", strings.NewReader("{}"))
	getReq.WithMeta("Authorization", "Bearer valid-token")
	getReq.WithMeta("user_id", "user_1")

	getResp, err := router.Do(getReq)
	if err != nil {
		log.Printf("Get error: %v", err)
		return
	}

	fmt.Printf("Get User - Status: %d\n", getResp.Status)
	fmt.Printf("User Data: %s", getResp.String())

	// Output:
	// Create User - Status: 201
	// Location Header: /users/user_1
	// Get User - Status: 200
	// User Data: {"id":"user_1","name":"John Doe","email":"john@example.com"}
}

// Real-world example: Message queue processor
type MessageProcessor struct {
	router *handlers.Router
}

type Message struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
	Meta    map[string]string      `json:"meta"`
}

func NewMessageProcessor() *MessageProcessor {
	mp := &MessageProcessor{
		router: handlers.NewRouter(),
	}

	// Add middleware
	mp.router.Use(handlers.LoggingMiddleware(func(pattern string, duration time.Duration, status int) {
		log.Printf("Processed %s in %v with status %d", pattern, duration, status)
	}))

	mp.router.Use(handlers.RecoveryMiddleware())

	// Register message handlers
	mp.router.HandleFunc("user.created", mp.handleUserCreated)
	mp.router.HandleFunc("order.placed", mp.handleOrderPlaced)
	mp.router.HandleFunc("email.send", mp.handleEmailSend)

	return mp
}

func (mp *MessageProcessor) handleUserCreated(w handlers.ResponseWriter, r *handlers.Request) error {
	var msg Message
	if err := r.Decode(&msg); err != nil {
		return err
	}

	// Process user creation
	userID, ok := msg.Payload["user_id"].(string)
	if !ok {
		w.WriteStatus(400)
		return w.Encode(map[string]string{"error": "invalid user_id"})
	}

	// Simulate processing
	log.Printf("Setting up user profile for %s", userID)
	log.Printf("Sending welcome email to user %s", userID)

	w.SetMeta("processed_at", time.Now().Format(time.RFC3339))
	return w.Encode(map[string]string{"status": "processed"})
}

func (mp *MessageProcessor) handleOrderPlaced(w handlers.ResponseWriter, r *handlers.Request) error {
	var msg Message
	if err := r.Decode(&msg); err != nil {
		return err
	}

	// Process order
	orderID, ok := msg.Payload["order_id"].(string)
	if !ok {
		w.WriteStatus(400)
		return w.Encode(map[string]string{"error": "invalid order_id"})
	}

	// Simulate processing
	log.Printf("Processing payment for order %s", orderID)
	log.Printf("Updating inventory for order %s", orderID)
	log.Printf("Sending confirmation email for order %s", orderID)

	return w.Encode(map[string]string{"status": "processed"})
}

func (mp *MessageProcessor) handleEmailSend(w handlers.ResponseWriter, r *handlers.Request) error {
	var msg Message
	if err := r.Decode(&msg); err != nil {
		return err
	}

	// Extract email details
	recipient, _ := msg.Payload["recipient"].(string)
	subject, _ := msg.Payload["subject"].(string)

	// Simulate email sending
	log.Printf("Sending email to %s with subject: %s", recipient, subject)

	return w.Encode(map[string]string{"status": "sent"})
}

func (mp *MessageProcessor) ProcessMessage(msgType string, payload map[string]interface{}) error {
	message := Message{
		Type:    msgType,
		Payload: payload,
		Meta:    map[string]string{"timestamp": time.Now().Format(time.RFC3339)},
	}

	// Convert message to JSON
	body := strings.NewReader(fmt.Sprintf(`{"type":"%s","payload":%s}`,
		message.Type, toJSON(message.Payload)))

	req := handlers.NewRequest(msgType, body)
	for k, v := range message.Meta {
		req.WithMeta(k, v)
	}

	_, err := mp.router.Do(req)
	return err
}

func toJSON(v interface{}) string {
	// Simple JSON conversion for the example
	return "{}"
}

func ExampleMessageProcessor() {
	processor := NewMessageProcessor()

	// Process different types of messages
	messages := []struct {
		msgType string
		payload map[string]interface{}
	}{
		{
			msgType: "user.created",
			payload: map[string]interface{}{"user_id": "user_123"},
		},
		{
			msgType: "order.placed",
			payload: map[string]interface{}{"order_id": "order_456"},
		},
		{
			msgType: "email.send",
			payload: map[string]interface{}{
				"recipient": "user@example.com",
				"subject":   "Welcome to our service!",
			},
		},
	}

	for _, msg := range messages {
		if err := processor.ProcessMessage(msg.msgType, msg.payload); err != nil {
			log.Printf("Failed to process message %s: %v", msg.msgType, err)
		}
	}

	fmt.Println("All messages processed")
	// Output: All messages processed
}

// Real-world example: API testing framework
type APITest struct {
	router *handlers.Router
}

func NewAPITest() *APITest {
	return &APITest{
		router: handlers.NewRouter(),
	}
}

func (at *APITest) RegisterEndpoint(pattern string, handler handlers.HandlerFunc) {
	at.router.HandleFunc(pattern, handler)
}

func (at *APITest) GET(pattern string, meta map[string]string) (*handlers.Response, error) {
	req := handlers.NewRequest(pattern, strings.NewReader("{}"))
	for k, v := range meta {
		req.WithMeta(k, v)
	}
	return at.router.Do(req)
}

func (at *APITest) POST(pattern string, body string, meta map[string]string) (*handlers.Response, error) {
	req := handlers.NewRequest(pattern, strings.NewReader(body))
	for k, v := range meta {
		req.WithMeta(k, v)
	}
	return at.router.Do(req)
}

func ExampleAPITest() {
	test := NewAPITest()

	// Register test endpoints
	test.RegisterEndpoint("api.health", func(w handlers.ResponseWriter, r *handlers.Request) error {
		return w.Encode(map[string]string{"status": "healthy"})
	})

	test.RegisterEndpoint("api.echo", func(w handlers.ResponseWriter, r *handlers.Request) error {
		var body map[string]interface{}
		if err := r.Decode(&body); err != nil {
			w.WriteStatus(400)
			return w.Encode(map[string]string{"error": "invalid JSON"})
		}
		return w.Encode(body)
	})

	// Test health endpoint
	resp, err := test.GET("api.health", nil)
	if err != nil {
		log.Printf("Health check failed: %v", err)
		return
	}
	fmt.Printf("Health check - Status: %d, Body: %s", resp.Status, resp.String())

	// Test echo endpoint
	resp, err = test.POST("api.echo", `{"message":"hello world"}`, nil)
	if err != nil {
		log.Printf("Echo test failed: %v", err)
		return
	}
	fmt.Printf("Echo test - Status: %d, Body: %s", resp.Status, resp.String())

	// Output:
	// Health check - Status: 200, Body: {"status":"healthy"}
	// Echo test - Status: 200, Body: {"message":"hello world"}
}

// Example: Request with timeout and context
func ExampleRouter_timeout() {
	router := handlers.NewRouter().WithTimeout(100 * time.Millisecond)

	// Handler that takes too long
	router.HandleFunc("slow", func(w handlers.ResponseWriter, r *handlers.Request) error {
		time.Sleep(200 * time.Millisecond) // Longer than timeout
		return w.Encode(map[string]string{"status": "done"})
	})

	// Handler that completes quickly
	router.HandleFunc("fast", func(w handlers.ResponseWriter, r *handlers.Request) error {
		return w.Encode(map[string]string{"status": "done"})
	})

	// Test slow handler
	req := handlers.NewRequest("slow", strings.NewReader("{}"))
	resp, err := router.Do(req)
	if err != nil {
		fmt.Printf("Slow handler error: %v\n", err)
	}
	if resp != nil {
		fmt.Printf("Slow handler status: %d\n", resp.Status)
	}

	// Test fast handler
	req = handlers.NewRequest("fast", strings.NewReader("{}"))
	resp, err = router.Do(req)
	if err != nil {
		fmt.Printf("Fast handler error: %v\n", err)
	} else {
		fmt.Printf("Fast handler status: %d\n", resp.Status)
	}

	// Output:
	// Slow handler error: handlers: request timeout
	// Fast handler status: 200
}

// Example: Custom context usage
func ExampleRequest_context() {
	router := handlers.NewRouter()

	router.HandleFunc("context-demo", func(w handlers.ResponseWriter, r *handlers.Request) error {
		// Check for user ID in context
		if userID := r.Context().Value("user_id"); userID != nil {
			return w.Encode(map[string]interface{}{
				"message": "Hello user",
				"user_id": userID,
			})
		}

		return w.Encode(map[string]string{"message": "Hello anonymous"})
	})

	// Request without context
	req := handlers.NewRequest("context-demo", strings.NewReader("{}"))
	resp, _ := router.Do(req)
	fmt.Printf("Without context: %s", resp.String())

	// Request with context
	ctx := context.WithValue(context.Background(), "user_id", "user_123")
	req = handlers.NewRequest("context-demo", strings.NewReader("{}")).WithContext(ctx)
	resp, _ = router.Do(req)
	fmt.Printf("With context: %s", resp.String())

	// Output:
	// Without context: {"message":"Hello anonymous"}
	// With context: {"message":"Hello user","user_id":"user_123"}
}
