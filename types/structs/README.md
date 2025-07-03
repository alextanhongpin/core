# Structs

The `structs` package provides comprehensive utilities for struct reflection, introspection, validation, and manipulation in Go. It offers type-safe operations for analyzing struct fields, tags, and values, making it easier to work with structs dynamically.

## Features

- **Type Introspection**: Get detailed type information including package names and kinds
- **Field Analysis**: Access field names, values, and existence checks
- **Struct Tag Support**: Extract and analyze struct tags for any key
- **Validation**: Validate that all struct fields contain non-zero values
- **Cloning**: Deep copy structs using JSON serialization
- **Nil Checking**: Safe nil checking for pointers and interface types
- **Type Safety**: All operations are type-safe with proper error handling

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/alextanhongpin/core/types/structs"
)

type User struct {
    ID    int    `json:"id" validate:"required"`
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
}

func main() {
    user := User{ID: 1, Name: "Alice", Email: "alice@example.com"}
    
    // Type introspection
    fmt.Println("Type:", structs.Type(user))     // Type: main.User
    fmt.Println("Name:", structs.Name(user))     // Name: User
    fmt.Println("Kind:", structs.Kind(user))     // Kind: struct
    
    // Field operations
    names, _ := structs.GetFieldNames(user)
    fmt.Println("Fields:", names)                // Fields: [ID Name Email]
    
    // Validation
    if err := structs.NonZero(user); err != nil {
        fmt.Println("Validation failed:", err)
    } else {
        fmt.Println("All fields are valid!")    // All fields are valid!
    }
}
```

## API Reference

### Type Introspection

```go
user := User{ID: 1, Name: "Alice"}

// Get type information
fullType := structs.Type(user)           // "main.User"
pkgName := structs.PkgName(user)         // "main.User" 
simpleName := structs.Name(user)         // "User"
kind := structs.Kind(user)               // reflect.Struct

// Type checking
isStruct := structs.IsStruct(user)       // true
isPointer := structs.IsPointer(&user)    // true
isNil := structs.IsNil((*User)(nil))     // true
```

### Field Operations

```go
// Check field existence
hasName := structs.HasField(user, "Name")        // true
hasPassword := structs.HasField(user, "Password") // false

// Get field names
names, err := structs.GetFieldNames(user)        // ["ID", "Name", "Email"]

// Get specific field value
name, err := structs.GetFieldValue(user, "Name") // "Alice"

// Get all fields as map
fields, err := structs.GetFields(user)           // map[string]any
```

### Struct Tags

```go
// Get tags by key
jsonTags, err := structs.GetTags(user, "json")      // map[field]tag
validateTags, err := structs.GetTags(user, "validate") // map[field]tag

// Example output:
// jsonTags: {"ID": "id", "Name": "name", "Email": "email"}
// validateTags: {"ID": "required", "Name": "required", "Email": "required,email"}
```

### Validation

```go
// Validate all fields are non-zero
err := structs.NonZero(user)
if err != nil {
    var fieldErr *structs.FieldError
    if errors.As(err, &fieldErr) {
        fmt.Printf("Empty field: %s (path: %s)\n", fieldErr.Field, fieldErr.Path)
    }
}
```

### Cloning

```go
// Deep clone a struct
original := User{ID: 1, Name: "Alice"}
cloned, err := structs.Clone(original)

// Modifications to original don't affect clone
original.Name = "Bob"
fmt.Println(cloned.Name) // Still "Alice"
```

## Real-World Examples

### Configuration Validation

```go
type DatabaseConfig struct {
    Host     string `json:"host" validate:"required"`
    Port     int    `json:"port" validate:"required,min=1,max=65535"`
    Username string `json:"username" validate:"required"`
    Password string `json:"password" validate:"required"`
    Database string `json:"database" validate:"required"`
    SSL      bool   `json:"ssl"`
}

type Config struct {
    Database DatabaseConfig `json:"database"`
    Redis    struct {
        Host string `json:"host" validate:"required"`
        Port int    `json:"port" validate:"required"`
    } `json:"redis"`
    Server struct {
        Port int    `json:"port" validate:"required"`
        Host string `json:"host"`
    } `json:"server"`
}

func ValidateConfig(config Config) error {
    // Validate all required fields are present
    if err := structs.NonZero(config); err != nil {
        var fieldErr *structs.FieldError
        if errors.As(err, &fieldErr) {
            return fmt.Errorf("missing required configuration: %s", fieldErr.Field)
        }
        return err
    }
    
    // Additional validation can be added here
    return nil
}

// Usage
config := Config{
    Database: DatabaseConfig{
        Host:     "localhost",
        Port:     5432,
        Username: "admin",
        Password: "secret",
        Database: "myapp",
        SSL:      true,
    },
}
config.Redis.Host = "localhost"
config.Redis.Port = 6379
config.Server.Port = 8080

if err := ValidateConfig(config); err != nil {
    log.Fatal("Configuration error:", err)
}
```

### Dynamic API Response Processing

```go
type APIResponse struct {
    Success bool                   `json:"success"`
    Data    map[string]interface{} `json:"data,omitempty"`
    Error   string                 `json:"error,omitempty"`
    Meta    struct {
        Page      int `json:"page"`
        PerPage   int `json:"per_page"`
        Total     int `json:"total"`
        TotalPages int `json:"total_pages"`
    } `json:"meta,omitempty"`
}

func ProcessAPIResponse(response APIResponse) error {
    // Check response structure
    fmt.Printf("Response type: %s\n", structs.Name(response))
    
    // Get all non-empty fields
    fields, err := structs.GetFields(response)
    if err != nil {
        return err
    }
    
    // Process based on available fields
    if success, ok := fields["success"].(bool); ok {
        if success {
            fmt.Println("✓ API call successful")
            
            // Process data if present
            if _, hasData := fields["data"]; hasData {
                fmt.Println("✓ Response contains data")
            }
            
            // Process pagination if present
            if _, hasMeta := fields["meta"]; hasMeta {
                fmt.Println("✓ Response contains pagination info")
            }
        } else {
            // Check for error message
            if errorMsg, hasError := fields["error"].(string); hasError {
                return fmt.Errorf("API error: %s", errorMsg)
            }
            return fmt.Errorf("API call failed with no error message")
        }
    }
    
    return nil
}
```

### Struct Tag-Based Serialization

```go
type Product struct {
    ID          int     `json:"id" db:"product_id" csv:"ID"`
    Name        string  `json:"name" db:"product_name" csv:"Name"`
    Price       float64 `json:"price" db:"price" csv:"Price"`
    Description string  `json:"description" db:"description" csv:"Description"`
    InStock     bool    `json:"in_stock" db:"in_stock" csv:"In Stock"`
    CreatedAt   time.Time `json:"created_at" db:"created_at" csv:"Created"`
}

func GetFieldMapping(product Product, format string) (map[string]string, error) {
    // Get field mappings for different formats
    tags, err := structs.GetTags(product, format)
    if err != nil {
        return nil, err
    }
    
    // Create field name to tag mapping
    mapping := make(map[string]string)
    fieldNames, _ := structs.GetFieldNames(product)
    
    for _, fieldName := range fieldNames {
        if tag, exists := tags[fieldName]; exists {
            mapping[fieldName] = tag
        } else {
            mapping[fieldName] = fieldName // Use field name as fallback
        }
    }
    
    return mapping, nil
}

// Usage
product := Product{ID: 1, Name: "Laptop", Price: 999.99}

// Get JSON field mappings
jsonMapping, _ := GetFieldMapping(product, "json")
// Result: {"ID": "id", "Name": "name", "Price": "price", ...}

// Get database column mappings  
dbMapping, _ := GetFieldMapping(product, "db")
// Result: {"ID": "product_id", "Name": "product_name", ...}

// Get CSV header mappings
csvMapping, _ := GetFieldMapping(product, "csv")
// Result: {"ID": "ID", "Name": "Name", "InStock": "In Stock", ...}
```

### Form Validation with Error Details

```go
type UserForm struct {
    FirstName string `json:"first_name" validate:"required,min=2"`
    LastName  string `json:"last_name" validate:"required,min=2"`
    Email     string `json:"email" validate:"required,email"`
    Age       int    `json:"age" validate:"min=13,max=120"`
    Terms     bool   `json:"terms" validate:"required"`
}

type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Value   any    `json:"value"`
}

func ValidateUserForm(form UserForm) []ValidationError {
    var errors []ValidationError
    
    // Check for empty required fields
    if err := structs.NonZero(form); err != nil {
        var fieldErr *structs.FieldError
        if errors.As(err, &fieldErr) {
            errors = append(errors, ValidationError{
                Field:   fieldErr.Field,
                Message: "This field is required",
                Value:   nil,
            })
        }
    }
    
    // Get validation rules from tags
    rules, _ := structs.GetTags(form, "validate")
    
    // Validate each field against its rules
    for fieldName, rule := range rules {
        value, _ := structs.GetFieldValue(form, fieldName)
        
        // Simple validation logic (in real apps, use a validation library)
        if strings.Contains(rule, "email") {
            if email, ok := value.(string); ok {
                if !strings.Contains(email, "@") {
                    errors = append(errors, ValidationError{
                        Field:   fieldName,
                        Message: "Must be a valid email address",
                        Value:   email,
                    })
                }
            }
        }
        
        if strings.Contains(rule, "min=") {
            // Extract min value and validate
            // Implementation details omitted for brevity
        }
    }
    
    return errors
}
```

### Object Builder with Validation

```go
type ProductBuilder struct {
    product Product
}

func NewProductBuilder() *ProductBuilder {
    return &ProductBuilder{product: Product{}}
}

func (pb *ProductBuilder) WithID(id int) *ProductBuilder {
    pb.product.ID = id
    return pb
}

func (pb *ProductBuilder) WithName(name string) *ProductBuilder {
    pb.product.Name = name
    return pb
}

func (pb *ProductBuilder) WithPrice(price float64) *ProductBuilder {
    pb.product.Price = price
    return pb
}

func (pb *ProductBuilder) Build() (Product, error) {
    // Validate that all required fields are set
    if err := structs.NonZero(pb.product); err != nil {
        var fieldErr *structs.FieldError
        if errors.As(err, &fieldErr) {
            return Product{}, fmt.Errorf("missing required field: %s", fieldErr.Field)
        }
        return Product{}, err
    }
    
    // Additional business logic validation
    if pb.product.Price <= 0 {
        return Product{}, fmt.Errorf("price must be positive")
    }
    
    return pb.product, nil
}

// Usage
product, err := NewProductBuilder().
    WithID(1).
    WithName("Gaming Laptop").
    WithPrice(1299.99).
    Build()

if err != nil {
    fmt.Printf("Failed to build product: %v\n", err)
} else {
    fmt.Printf("Created product: %+v\n", product)
}
```

### Dynamic Struct Copying

```go
func CopyNonZeroFields(src, dst any) error {
    if !structs.IsStruct(src) || !structs.IsStruct(dst) {
        return fmt.Errorf("both src and dst must be structs")
    }
    
    // Get source fields
    srcFields, err := structs.GetFields(src)
    if err != nil {
        return err
    }
    
    // Get destination field names to check compatibility
    dstNames, err := structs.GetFieldNames(dst)
    if err != nil {
        return err
    }
    
    dstNameSet := make(map[string]bool)
    for _, name := range dstNames {
        dstNameSet[name] = true
    }
    
    // Copy non-zero fields that exist in destination
    srcNames, _ := structs.GetFieldNames(src)
    for _, fieldName := range srcNames {
        if !dstNameSet[fieldName] {
            continue // Skip fields that don't exist in destination
        }
        
        srcValue, err := structs.GetFieldValue(src, fieldName)
        if err != nil {
            continue
        }
        
        // Check if source field is non-zero
        if !structs.isEmpty(srcValue) {
            // In a real implementation, you'd use reflection to set the field
            fmt.Printf("Would copy %s: %v\n", fieldName, srcValue)
        }
    }
    
    return nil
}
```

### Audit Logging

```go
type AuditEntry struct {
    Timestamp time.Time              `json:"timestamp"`
    Action    string                 `json:"action"`
    Entity    string                 `json:"entity"`
    EntityID  string                 `json:"entity_id"`
    Changes   map[string]interface{} `json:"changes"`
    User      string                 `json:"user"`
}

func CreateAuditLog(entity any, action, user string) AuditEntry {
    entry := AuditEntry{
        Timestamp: time.Now(),
        Action:    action,
        Entity:    structs.Name(entity),
        User:      user,
    }
    
    // Get entity ID if it has one
    if id, err := structs.GetFieldValue(entity, "ID"); err == nil {
        entry.EntityID = fmt.Sprintf("%v", id)
    }
    
    // Capture all field values as changes
    if fields, err := structs.GetFields(entity); err == nil {
        entry.Changes = fields
    }
    
    return entry
}

// Usage
user := User{ID: 1, Name: "Alice", Email: "alice@example.com"}
auditEntry := CreateAuditLog(user, "CREATE", "admin")
fmt.Printf("Audit: %+v\n", auditEntry)
```

## Best Practices

1. **Validation**: Use `NonZero()` for basic required field validation
2. **Error Handling**: Always check for `FieldError` when validation fails
3. **Performance**: Cache reflection results for frequently accessed structs
4. **Type Safety**: Verify struct types before performing operations
5. **Tag Conventions**: Use consistent tag naming across your application
6. **Clone Limitations**: JSON-based cloning only works with JSON-serializable fields

## Performance Considerations

- **Reflection Cost**: Struct introspection uses reflection, which has overhead
- **JSON Cloning**: `Clone()` uses JSON marshaling, which may be slower than manual copying
- **Caching**: Consider caching type information for frequently accessed structs
- **Field Access**: Direct field access is faster than `GetFieldValue()` when possible

## Limitations

1. **Private Fields**: Only exported (public) fields are accessible
2. **JSON Cloning**: `Clone()` only copies JSON-serializable fields
3. **Type Conversions**: Some operations require type assertions
4. **Reflection Overhead**: Performance impact for high-frequency operations

## Thread Safety

The structs package functions are read-only and thread-safe for concurrent access to the same struct values. However, modifying struct values while performing operations is not safe without external synchronization.

## Integration Examples

### With Validation Libraries

```go
import "github.com/go-playground/validator/v10"

func ValidateStruct(s any) error {
    // First check for non-zero fields
    if err := structs.NonZero(s); err != nil {
        return fmt.Errorf("required field validation failed: %w", err)
    }
    
    // Then use a proper validation library
    validate := validator.New()
    return validate.Struct(s)
}
```

### With ORM Integration

```go
func GetDBColumnName(entity any, fieldName string) (string, error) {
    dbTags, err := structs.GetTags(entity, "db")
    if err != nil {
        return "", err
    }
    
    if columnName, exists := dbTags[fieldName]; exists {
        return columnName, nil
    }
    
    return fieldName, nil // Default to field name
}
```
