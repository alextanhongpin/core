package assert_test

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/alextanhongpin/core/types/assert"
)

// Example: User Registration API
type UserRegistrationRequest struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	Age          int    `json:"age"`
	Country      string `json:"country"`
	PhoneNumber  string `json:"phone_number,omitempty"`
	ReferralCode string `json:"referral_code,omitempty"`
}

func (r *UserRegistrationRequest) Validate() map[string]string {
	return assert.Map(map[string]string{
		"username": assert.Required(r.Username,
			assert.MinLength(r.Username, 3),
			assert.MaxLength(r.Username, 20),
		),
		"email": assert.Required(r.Email,
			assert.Email(r.Email),
		),
		"password": assert.Required(r.Password,
			assert.MinLength(r.Password, 8),
			assert.MaxLength(r.Password, 128),
		),
		"age": assert.Required(r.Age,
			assert.Range(r.Age, 13, 120),
		),
		"country": assert.Required(r.Country,
			assert.OneOf(r.Country, "US", "CA", "MX", "GB", "DE", "FR", "JP", "SG", "MY"),
		),
		"phone_number": assert.Optional(r.PhoneNumber,
			assert.MinLength(r.PhoneNumber, 10),
		),
		"referral_code": assert.Optional(r.ReferralCode,
			assert.MinLength(r.ReferralCode, 6),
			assert.MaxLength(r.ReferralCode, 12),
		),
	})
}

func ExampleUserRegistrationRequest() {
	// Simulate an HTTP handler for user registration
	handler := func(w http.ResponseWriter, r *http.Request) {
		var req UserRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validate the request
		if errors := req.Validate(); len(errors) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":  "Validation failed",
				"fields": errors,
			})
			return
		}

		// Process successful registration...
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "User registered successfully",
			"user_id": "user_12345",
		})
	}

	// This would be used in your HTTP server
	_ = handler
}

// Example: Configuration Validation
type DatabaseConfig struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Database        string `json:"database"`
	MaxConnections  int    `json:"max_connections"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	SSLMode         string `json:"ssl_mode"`
	BackupRetention int    `json:"backup_retention,omitempty"`
}

func (c *DatabaseConfig) Validate() map[string]string {
	return assert.Map(map[string]string{
		"host": assert.Required(c.Host),
		"port": assert.Required(c.Port,
			assert.Range(c.Port, 1, 65535),
		),
		"username": assert.Required(c.Username,
			assert.MinLength(c.Username, 1),
		),
		"password": assert.Required(c.Password,
			assert.MinLength(c.Password, 8),
		),
		"database": assert.Required(c.Database,
			assert.MinLength(c.Database, 1),
		),
		"max_connections": assert.Required(c.MaxConnections,
			assert.Range(c.MaxConnections, 1, 1000),
		),
		"timeout_seconds": assert.Required(c.TimeoutSeconds,
			assert.Range(c.TimeoutSeconds, 1, 300),
		),
		"ssl_mode": assert.Required(c.SSLMode,
			assert.OneOf(c.SSLMode, "disable", "require", "verify-ca", "verify-full"),
		),
		"backup_retention": assert.Optional(c.BackupRetention,
			assert.Range(c.BackupRetention, 1, 365),
		),
	})
}

func ExampleDatabaseConfig_Validate() {
	config := DatabaseConfig{
		Host:           "localhost",
		Port:           5432,
		Username:       "admin",
		Password:       "secretpassword123",
		Database:       "myapp",
		MaxConnections: 100,
		TimeoutSeconds: 30,
		SSLMode:        "require",
	}

	if errors := config.Validate(); len(errors) > 0 {
		log.Printf("Configuration validation failed: %+v", errors)
		return
	}

	log.Println("Configuration is valid")
}

// Example: E-commerce Order Validation
type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

func (item *OrderItem) Validate() map[string]string {
	return assert.Map(map[string]string{
		"product_id": assert.Required(item.ProductID,
			assert.MinLength(item.ProductID, 1),
		),
		"quantity": assert.Required(item.Quantity,
			assert.Range(item.Quantity, 1, 100),
		),
		"price": assert.Required(item.Price,
			assert.Is(item.Price > 0, "must be greater than 0"),
		),
	})
}

type Order struct {
	CustomerID   string      `json:"customer_id"`
	Items        []OrderItem `json:"items"`
	DiscountCode string      `json:"discount_code,omitempty"`
	TotalAmount  float64     `json:"total_amount"`
	Currency     string      `json:"currency"`
}

func (o *Order) Validate() map[string]string {
	result := map[string]string{
		"customer_id": assert.Required(o.CustomerID),
		"items":       assert.Required(len(o.Items)),
		"total_amount": assert.Required(o.TotalAmount,
			assert.Is(o.TotalAmount > 0, "must be greater than 0"),
		),
		"currency": assert.Required(o.Currency,
			assert.OneOf(o.Currency, "USD", "EUR", "GBP", "JPY", "CAD", "AUD"),
		),
		"discount_code": assert.Optional(o.DiscountCode,
			assert.MinLength(o.DiscountCode, 3),
			assert.MaxLength(o.DiscountCode, 20),
		),
	}

	// Validate each item
	for i, item := range o.Items {
		for field, err := range item.Validate() {
			result[fmt.Sprintf("items[%d].%s", i, field)] = err
		}
	}

	return assert.Map(result)
}

func ExampleOrder_Validate() {
	order := Order{
		CustomerID: "cust_12345",
		Items: []OrderItem{
			{ProductID: "prod_1", Quantity: 2, Price: 29.99},
			{ProductID: "", Quantity: 0, Price: -10}, // Invalid item
		},
		TotalAmount: 59.98,
		Currency:    "USD",
	}

	if errors := order.Validate(); len(errors) > 0 {
		fmt.Println("Order validation errors:")
		for field, err := range errors {
			fmt.Printf("  %s: %s\n", field, err)
		}
	}
	// Output:
	// Order validation errors:
	//   items[1].price: must be greater than 0
	//   items[1].product_id: required
	//   items[1].quantity: must be between 1 and 100
}
