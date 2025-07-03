# Assert - Structured Validation Library

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/types/assert.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/types/assert)

Package `assert` provides utilities for building structured validation systems in Go. It enables creating composable validation logic with clear, structured error reporting - perfect for API request validation, configuration validation, and other scenarios where detailed error feedback is essential.

## Features

- **Composable Validation**: Build complex validation rules by combining simple validators
- **Structured Error Reporting**: Generate field-specific error messages in a map format
- **Required vs Optional**: Different validation behavior for required and optional fields
- **Built-in Validators**: Common validation functions for strings, numbers, emails, etc.
- **Zero Dependencies**: Pure Go implementation with no external dependencies
- **Type Safe**: Leverages Go generics for type-safe validation functions

## Installation

```bash
go get github.com/alextanhongpin/core/types/assert
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/alextanhongpin/core/types/assert"
)

type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

func (u *User) Validate() map[string]string {
    return assert.Map(map[string]string{
        "name": assert.Required(u.Name,
            assert.MinLength(u.Name, 2),
            assert.MaxLength(u.Name, 50),
        ),
        "email": assert.Required(u.Email,
            assert.Email(u.Email),
        ),
        "age": assert.Required(u.Age,
            assert.Range(u.Age, 18, 120),
        ),
    })
}

func main() {
    user := User{Name: "A", Email: "invalid", Age: 15}
    
    if errors := user.Validate(); len(errors) > 0 {
        fmt.Printf("Validation errors: %+v\n", errors)
        // Output: map[age:must be between 18 and 120 email:must be a valid email address name:must be at least 2 characters]
    }
}
```

## Core Functions

### Required vs Optional

- **`Required(v any, assertions ...string)`**: Validates that a value is non-zero and applies additional assertions
- **`Optional(v any, assertions ...string)`**: Applies assertions only if the value is non-zero

```go
// Required field - will fail if empty or invalid
password := assert.Required(user.Password,
    assert.MinLength(user.Password, 8),
)

// Optional field - only validates if not empty
middleName := assert.Optional(user.MiddleName,
    assert.MinLength(user.MiddleName, 2),
)
```

### Built-in Validators

- **`Is(condition bool, message string)`**: Conditional validation
- **`MinLength(s string, min int)`**: Minimum string length
- **`MaxLength(s string, max int)`**: Maximum string length
- **`Range[T](v, min, max T)`**: Numeric/string range validation
- **`Email(s string)`**: Email format validation
- **`OneOf[T](v T, allowed ...T)`**: Whitelist validation

## Real-World Examples

### API Request Validation

```go
type CreateUserRequest struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
    Country  string `json:"country"`
}

func (r *CreateUserRequest) Validate() map[string]string {
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
        ),
        "country": assert.Required(r.Country,
            assert.OneOf(r.Country, "US", "CA", "GB", "DE", "FR"),
        ),
    })
}
```

### Configuration Validation

```go
type DatabaseConfig struct {
    Host     string `yaml:"host"`
    Port     int    `yaml:"port"`
    Username string `yaml:"username"`
    Password string `yaml:"password"`
    SSLMode  string `yaml:"ssl_mode"`
}

func (c *DatabaseConfig) Validate() map[string]string {
    return assert.Map(map[string]string{
        "host": assert.Required(c.Host),
        "port": assert.Required(c.Port,
            assert.Range(c.Port, 1, 65535),
        ),
        "username": assert.Required(c.Username),
        "password": assert.Required(c.Password,
            assert.MinLength(c.Password, 8),
        ),
        "ssl_mode": assert.Required(c.SSLMode,
            assert.OneOf(c.SSLMode, "disable", "require", "verify-ca"),
        ),
    })
}
```

### Nested Validation

```go
type Order struct {
    CustomerID string      `json:"customer_id"`
    Items      []OrderItem `json:"items"`
    Total      float64     `json:"total"`
}

type OrderItem struct {
    ProductID string  `json:"product_id"`
    Quantity  int     `json:"quantity"`
    Price     float64 `json:"price"`
}

func (o *Order) Validate() map[string]string {
    result := map[string]string{
        "customer_id": assert.Required(o.CustomerID),
        "items":       assert.Required(len(o.Items)),
        "total": assert.Required(o.Total,
            assert.Is(o.Total > 0, "must be greater than 0"),
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

func (item *OrderItem) Validate() map[string]string {
    return assert.Map(map[string]string{
        "product_id": assert.Required(item.ProductID),
        "quantity": assert.Required(item.Quantity,
            assert.Range(item.Quantity, 1, 100),
        ),
        "price": assert.Required(item.Price,
            assert.Is(item.Price > 0, "must be greater than 0"),
        ),
    })
}
```

## Integration with HTTP APIs

```go
func createUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
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

    // Process valid request...
}
```

## Best Practices

1. **Return Empty Maps for Valid Data**: Use `assert.Map()` to filter out empty validation messages
2. **Compose Validators**: Combine multiple validators for comprehensive validation
3. **Use Required/Optional Appropriately**: Be explicit about field requirements
4. **Validate Early**: Validate input at system boundaries (HTTP handlers, config loading)
5. **Structured Errors**: Return field-specific errors for better UX

## Custom Validators

Create custom validators by following the same pattern:

```go
func ValidAge(age int) string {
    return assert.Is(age >= 18 && age <= 120, "must be between 18 and 120")
}

// Usage
"age": assert.Required(user.Age, ValidAge(user.Age))
```

## Architecture Decision Records

For detailed design decisions and rationale, see: https://github.com/alextanhongpin/architecture-decision-records/blob/master/golang/066-validation-errors.md

## License

MIT License - see LICENSE file for details.
