package structs_test

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/alextanhongpin/core/types/structs"
	"github.com/stretchr/testify/assert"
)

// Example types for testing
type User struct {
	ID        int       `json:"id" db:"user_id" validate:"required"`
	Name      string    `json:"name" db:"full_name" validate:"required,min=2"`
	Email     string    `json:"email" db:"email_address" validate:"required,email"`
	Age       int       `json:"age" db:"age" validate:"min=0,max=150"`
	Active    bool      `json:"active" db:"is_active"`
	Profile   *Profile  `json:"profile,omitempty"`
	Metadata  Metadata  `json:"metadata"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Profile struct {
	Bio      string `json:"bio"`
	Website  string `json:"website"`
	Location string `json:"location"`
	Avatar   string `json:"avatar"`
}

type Metadata map[string]any

type Product struct {
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	InStock     bool    `json:"in_stock"`
}

// Example: Type Introspection
func ExampleType_typeIntrospection() {
	user := User{ID: 1, Name: "Alice"}
	userPtr := &user

	fmt.Printf("User type: %s\n", structs.Type(user))
	fmt.Printf("User pointer type: %s\n", structs.Type(userPtr))
	fmt.Printf("User package name: %s\n", structs.PkgName(user))
	fmt.Printf("User simple name: %s\n", structs.Name(user))
	fmt.Printf("User kind: %s\n", structs.Kind(user))
	fmt.Printf("Is struct: %v\n", structs.IsStruct(user))
	fmt.Printf("Is pointer: %v\n", structs.IsPointer(userPtr))

	// Output:
	// User type: structs_test.User
	// User pointer type: *structs_test.User
	// User package name: structs_test.User
	// User simple name: User
	// User kind: struct
	// Is struct: true
	// Is pointer: true
}

// Example: Field Analysis
func ExampleGetFields_fieldAnalysis() {
	user := User{
		ID:     1,
		Name:   "Alice",
		Email:  "alice@example.com",
		Age:    30,
		Active: true,
		Tags:   []string{"admin", "premium"},
	}

	// Get all fields
	fields, err := structs.GetFields(user)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("User has %d fields\n", len(fields))

	// Get field names
	names, _ := structs.GetFieldNames(user)
	fmt.Printf("Field names: %v\n", names)

	// Check specific fields
	fmt.Printf("Has Name field: %v\n", structs.HasField(user, "Name"))
	fmt.Printf("Has Password field: %v\n", structs.HasField(user, "Password"))

	// Get specific field value
	if email, err := structs.GetFieldValue(user, "Email"); err == nil {
		fmt.Printf("Email: %s\n", email)
		if fields["Email"] == email {
			fmt.Println("Email field matches the value in fields map")
		} else {
			fmt.Println("Email field does not match the value in fields map")
		}
	} else {
		fmt.Println(err)
	}

	// Output:
	// User has 9 fields
	// Field names: [ID Name Email Age Active Profile Metadata Tags CreatedAt]
	// Has Name field: true
	// Has Password field: false
	// Email: alice@example.com
	// Email field matches the value in fields map
}

// Example: Struct Tag Analysis
func ExampleGetTags_tagAnalysis() {
	user := User{}

	// Get JSON tags
	jsonTags, err := structs.GetTags(user, "json")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("JSON tags:")
	fields := slices.Sorted(maps.Keys(jsonTags))
	for _, field := range fields {
		tag := jsonTags[field]
		fmt.Printf("  %s: %s\n", field, tag)
	}

	// Get validation tags
	validateTags, _ := structs.GetTags(user, "validate")
	fmt.Println("\nValidation tags:")

	fields = slices.Sorted(maps.Keys(validateTags))
	for _, field := range fields {
		tag := validateTags[field]
		fmt.Printf("  %s: %s\n", field, tag)
	}

	// Output:
	// JSON tags:
	//   Active: active
	//   Age: age
	//   CreatedAt: created_at
	//   Email: email
	//   ID: id
	//   Metadata: metadata
	//   Name: name
	//   Profile: profile,omitempty
	//   Tags: tags
	//
	// Validation tags:
	//   Age: min=0,max=150
	//   Email: required,email
	//   ID: required
	//   Name: required,min=2
}

// Example: Struct Validation
func ExampleNonZero_validation() {
	// Valid user
	validUser := User{
		ID:     1,
		Name:   "Alice",
		Email:  "alice@example.com",
		Age:    30,
		Active: true,
		Profile: &Profile{
			Bio:      "Software Engineer",
			Website:  "https://alice.dev",
			Avatar:   "https://alice.dev/avatar.png",
			Location: "Wonderland",
		},
		Metadata:  Metadata{"role": "admin"},
		Tags:      []string{"premium"},
		CreatedAt: time.Now(),
	}

	if err := structs.NonZero(validUser); err != nil {
		fmt.Printf("Valid user error: %v\n", err)
	} else {
		fmt.Println("Valid user: all fields non-zero ✓")
	}

	// Invalid user (missing name)
	invalidUser := User{
		ID:    1,
		Email: "alice@example.com",
		// Name is missing (empty string)
	}

	if err := structs.NonZero(invalidUser); err != nil {
		var fieldErr *structs.FieldError
		if errors.As(err, &fieldErr) {
			fmt.Printf("Invalid user - Field: %s, Path: %s\n",
				fieldErr.Field, fieldErr.Path)
		}
	}

	// Output:
	// Valid user: all fields non-zero ✓
	// Invalid user - Field: Name, Path: structs_test.User.Name
}

// Example: Struct Cloning
func ExampleClone_structCloning() {
	original := User{
		ID:    1,
		Name:  "Alice",
		Email: "alice@example.com",
		Profile: &Profile{
			Bio: "Original bio",
		},
		Tags: []string{"admin"},
	}

	// Clone the struct
	cloned, err := structs.Clone(original)
	if err != nil {
		fmt.Printf("Clone error: %v\n", err)
		return
	}

	// Modify the original
	original.Name = "Bob"
	original.Tags[0] = "user"

	fmt.Printf("Original name: %s\n", original.Name)
	fmt.Printf("Cloned name: %s\n", cloned.Name)
	fmt.Printf("Original tag: %s\n", original.Tags[0])
	fmt.Printf("Cloned tag: %s\n", cloned.Tags[0])

	// Note: Pointer fields are deeply cloned too
	if cloned.Profile != nil {
		fmt.Printf("Cloned profile bio: %s\n", cloned.Profile.Bio)
	}

	// Output:
	// Original name: Bob
	// Cloned name: Alice
	// Original tag: user
	// Cloned tag: admin
	// Cloned profile bio: Original bio
}

// Example: Type Checking and Nil Handling
func ExampleIsNil_typeChecking() {
	var user *User
	var iface any
	var slice []string
	var m map[string]int

	fmt.Printf("Nil pointer: %v\n", structs.IsNil(user))
	fmt.Printf("Nil interface: %v\n", structs.IsNil(iface))
	fmt.Printf("Nil slice: %v\n", structs.IsNil(slice))
	fmt.Printf("Nil map: %v\n", structs.IsNil(m))

	// Non-nil values
	user = &User{}
	slice = make([]string, 0)
	m = make(map[string]int)

	fmt.Printf("Empty struct pointer: %v\n", structs.IsNil(user))
	fmt.Printf("Empty slice: %v\n", structs.IsNil(slice))
	fmt.Printf("Empty map: %v\n", structs.IsNil(m))

	// Output:
	// Nil pointer: true
	// Nil interface: true
	// Nil slice: true
	// Nil map: true
	// Empty struct pointer: false
	// Empty slice: false
	// Empty map: false
}

// Example: Configuration Validation
func ExampleNonZero_configValidation() {
	type DatabaseConfig struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		Database string `json:"database"`
		SSL      bool   `json:"ssl"`
	}

	type ServerConfig struct {
		Database DatabaseConfig `json:"database"`
		Redis    struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"redis"`
		Port int `json:"port"`
	}

	// Valid configuration
	validConfig := ServerConfig{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Username: "admin",
			Password: "secret",
			Database: "myapp",
			SSL:      true,
		},
		Port: 8080,
	}

	// Add Redis config
	validConfig.Redis.Host = "localhost"
	validConfig.Redis.Port = 6379

	if err := structs.NonZero(validConfig); err != nil {
		fmt.Printf("Config validation failed: %v\n", err)
	} else {
		fmt.Println("Configuration is valid ✓")
	}

	// Invalid configuration (missing database password)
	invalidConfig := validConfig
	invalidConfig.Database.Password = ""

	if err := structs.NonZero(invalidConfig); err != nil {
		var fieldErr *structs.FieldError
		if errors.As(err, &fieldErr) {
			fmt.Printf("Missing required field: %s\n", fieldErr.Field)
		}
	}

	// Output:
	// Configuration is valid ✓
	// Missing required field: Password
}

// Example: API Response Processing
func ExampleGetFields_apiProcessing() {
	type APIResponse struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
		Error   string         `json:"error,omitempty"`
		Meta    struct {
			Page  int `json:"page"`
			Total int `json:"total"`
		} `json:"meta"`
	}

	response := APIResponse{
		Success: true,
		Data: map[string]any{
			"users": []map[string]any{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			},
		},
	}
	response.Meta.Page = 1
	response.Meta.Total = 2

	// Analyze response structure
	fields, err := structs.GetFields(response)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response type: %s\n", structs.Name(response))
	fmt.Printf("Has success field: %v\n", structs.HasField(response, "Success"))
	fmt.Printf("Has error field: %v\n", structs.HasField(response, "Error"))

	// Check if response indicates success
	if success, ok := fields["Success"].(bool); ok && success {
		fmt.Println("API call was successful")
	}

	// Output:
	// Response type: APIResponse
	// Has success field: true
	// Has error field: true
	// API call was successful
}

// Example: Struct Builder Pattern with Validation
func ExampleNonZero_builderPattern() {
	type UserBuilder struct {
		user User
	}

	newUserBuilder := func() *UserBuilder {
		return &UserBuilder{user: User{}}
	}

	withID := func(ub *UserBuilder, id int) *UserBuilder {
		ub.user.ID = id
		return ub
	}

	withName := func(ub *UserBuilder, name string) *UserBuilder {
		ub.user.Name = name
		return ub
	}

	withEmail := func(ub *UserBuilder, email string) *UserBuilder {
		ub.user.Email = email
		return ub
	}

	withTags := func(ub *UserBuilder, tags []string) *UserBuilder {
		ub.user.Tags = tags
		return ub
	}

	build := func(ub *UserBuilder) (User, error) {
		// Validate required fields before building
		if err := structs.NonZero(ub.user); err != nil {
			return User{}, fmt.Errorf("user validation failed: %w", err)
		}
		return ub.user, nil
	}

	// Build a valid user
	builder := newUserBuilder()
	builder.user.Age = 25                             // Optional field, can be set later
	builder.user.Active = true                        // Optional field, can be set later
	builder.user.Metadata = Metadata{"role": "admin"} // Optional field, can be set later

	builder = withID(builder, 1)
	builder = withName(builder, "Alice")
	builder = withEmail(builder, "alice@example.com")
	builder = withTags(builder, []string{"admin", "premium"})

	user, err := build(builder)
	if err != nil {
		fmt.Printf("Build failed: %v\n", err)
	} else {
		fmt.Printf("Built user: %s (%s)\n", user.Name, user.Email)
	}

	// Try to build an invalid user
	invalidBuilder := newUserBuilder()
	invalidBuilder = withID(invalidBuilder, 1)
	invalidBuilder = withName(invalidBuilder, "Bob")
	// Missing email

	_, err = build(invalidBuilder)
	if err != nil {
		fmt.Printf("Validation caught missing field: %v\n", err)
	}

	// Output:
	// Build failed: user validation failed: field "Profile" is empty
	// Validation caught missing field: user validation failed: field "Email" is empty
}

// Example: Dynamic Field Processing
func ExampleGetFieldValue_dynamicProcessing() {
	user := User{
		ID:    1,
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   30,
	}

	// Dynamic field processing based on field names
	fieldNames, _ := structs.GetFieldNames(user)

	fmt.Println("Processing user fields:")
	for _, fieldName := range fieldNames {
		value, err := structs.GetFieldValue(user, fieldName)
		if err != nil {
			continue
		}

		// Process different field types
		switch fieldName {
		case "ID":
			if id, ok := value.(int); ok && id > 0 {
				fmt.Printf("  Valid ID: %d\n", id)
			}
		case "Email":
			if email, ok := value.(string); ok && email != "" {
				fmt.Printf("  Valid email: %s\n", email)
			}
		case "Age":
			if age, ok := value.(int); ok && age >= 18 {
				fmt.Printf("  Adult user: %d years old\n", age)
			}
		}
	}

	// Output:
	// Processing user fields:
	//   Valid ID: 1
	//   Valid email: alice@example.com
	//   Adult user: 30 years old
}

// Unit Tests
func TestStructIntrospection(t *testing.T) {
	t.Run("Type detection", func(t *testing.T) {
		assert := assert.New(t)

		user := User{}
		userPtr := &user

		assert.Equal("structs_test.User", structs.Type(user))
		assert.Equal("*structs_test.User", structs.Type(userPtr))
		assert.Equal("User", structs.Name(user))
		assert.Equal("User", structs.Name(userPtr))
		assert.Equal(reflect.Struct, structs.Kind(user))
		assert.Equal(reflect.Struct, structs.Kind(userPtr))
	})

	t.Run("Struct validation", func(t *testing.T) {
		assert := assert.New(t)

		assert.True(structs.IsStruct(User{}))
		assert.True(structs.IsStruct(&User{}))
		assert.False(structs.IsStruct("string"))
		assert.False(structs.IsStruct(42))
	})

	t.Run("Nil checking", func(t *testing.T) {
		assert := assert.New(t)

		var user *User
		var iface any
		var slice []string

		assert.True(structs.IsNil(user))
		assert.True(structs.IsNil(iface))
		assert.True(structs.IsNil(slice))

		user = &User{}
		slice = make([]string, 0)

		assert.False(structs.IsNil(user))
		assert.False(structs.IsNil(slice))
	})
}

func TestFieldOperations(t *testing.T) {
	user := User{
		ID:    1,
		Name:  "Alice",
		Email: "alice@example.com",
	}

	t.Run("Field existence", func(t *testing.T) {
		assert := assert.New(t)

		assert.True(structs.HasField(user, "Name"))
		assert.True(structs.HasField(user, "Email"))
		assert.False(structs.HasField(user, "Password"))
	})

	t.Run("Field names", func(t *testing.T) {
		assert := assert.New(t)

		names, err := structs.GetFieldNames(user)
		assert.NoError(err)
		assert.Contains(names, "ID")
		assert.Contains(names, "Name")
		assert.Contains(names, "Email")
	})

	t.Run("Field values", func(t *testing.T) {
		assert := assert.New(t)

		name, err := structs.GetFieldValue(user, "Name")
		assert.NoError(err)
		assert.Equal("Alice", name)

		email, err := structs.GetFieldValue(user, "Email")
		assert.NoError(err)
		assert.Equal("alice@example.com", email)

		_, err = structs.GetFieldValue(user, "NonExistent")
		assert.Error(err)
	})

	t.Run("All fields", func(t *testing.T) {
		assert := assert.New(t)

		fields, err := structs.GetFields(user)
		assert.NoError(err)
		assert.Equal(1, fields["ID"]) // JSON unmarshaling converts int to float64
		assert.Equal("Alice", fields["Name"])
		assert.Equal("alice@example.com", fields["Email"])
	})
}

func TestStructTags(t *testing.T) {
	t.Run("JSON tags", func(t *testing.T) {
		assert := assert.New(t)

		user := User{}
		tags, err := structs.GetTags(user, "json")
		assert.NoError(err)

		assert.Equal("id", tags["ID"])
		assert.Equal("name", tags["Name"])
		assert.Equal("email", tags["Email"])
		assert.Equal("profile,omitempty", tags["Profile"])
	})

	t.Run("Validation tags", func(t *testing.T) {
		assert := assert.New(t)

		user := User{}
		tags, err := structs.GetTags(user, "validate")
		assert.NoError(err)

		assert.Equal("required", tags["ID"])
		assert.Equal("required,min=2", tags["Name"])
		assert.Equal("required,email", tags["Email"])
	})

	t.Run("Non-existent tags", func(t *testing.T) {
		assert := assert.New(t)

		user := User{}
		tags, err := structs.GetTags(user, "nonexistent")
		assert.NoError(err)
		assert.Empty(tags)
	})
}

func TestValidation(t *testing.T) {
	t.Run("Valid struct", func(t *testing.T) {
		assert := assert.New(t)

		user := User{
			ID:     1,
			Name:   "Alice",
			Email:  "alice@example.com",
			Age:    30,
			Active: true,
			Profile: &Profile{
				Avatar:   "https://alice.dev/avatar.png",
				Bio:      "Engineer",
				Website:  "https://alice.dev",
				Location: "Wonderland",
			},
			Metadata:  Metadata{"role": "admin"},
			Tags:      []string{"premium"},
			CreatedAt: time.Now(),
		}

		err := structs.NonZero(user)
		assert.NoError(err)
	})

	t.Run("Invalid struct - missing field", func(t *testing.T) {
		assert := assert.New(t)

		user := User{
			ID:    1,
			Email: "alice@example.com",
			// Name is missing
		}

		err := structs.NonZero(user)
		assert.Error(err)

		var fieldErr *structs.FieldError
		assert.True(errors.As(err, &fieldErr))
		assert.Equal("Name", fieldErr.Field)
	})

	t.Run("Invalid struct - empty nested struct", func(t *testing.T) {
		assert := assert.New(t)

		user := User{
			ID:        1,
			Name:      "Alice",
			Email:     "alice@example.com",
			Profile:   new(Profile), // Empty profile
			Active:    true,
			Tags:      []string{"admin"},
			CreatedAt: time.Now(),
			Age:       30,
			Metadata:  Metadata{"role": "admin"},
		}

		err := structs.NonZero(user)
		assert.Error(err)

		var fieldErr *structs.FieldError
		assert.True(errors.As(err, &fieldErr))
		assert.Equal(fieldErr.Path, "structs_test.User.Profile.Bio")
	})
}

func TestCloning(t *testing.T) {
	t.Run("Simple struct clone", func(t *testing.T) {
		assert := assert.New(t)

		original := Product{
			Name:        "Laptop",
			Price:       999.99,
			Description: "Gaming laptop",
			InStock:     true,
		}

		cloned, err := structs.Clone(original)
		assert.NoError(err)

		// Modify original
		original.Name = "Desktop"
		original.Price = 1299.99

		// Cloned should be unchanged
		assert.Equal("Laptop", cloned.Name)
		assert.Equal(999.99, cloned.Price)
	})

	t.Run("Complex struct clone", func(t *testing.T) {
		assert := assert.New(t)

		original := User{
			ID:   1,
			Name: "Alice",
			Profile: &Profile{
				Bio: "Engineer",
			},
			Tags: []string{"admin"},
		}

		cloned, err := structs.Clone(original)
		assert.NoError(err)

		// Modify original
		original.Name = "Bob"
		original.Tags[0] = "user"

		// Cloned should be unchanged
		assert.Equal("Alice", cloned.Name)
		assert.Equal([]string{"admin"}, cloned.Tags)
	})
}

// Benchmarks
func BenchmarkTypeIntrospection(b *testing.B) {
	user := User{ID: 1, Name: "Alice"}

	b.ResetTimer()
	for b.Loop() {
		_ = structs.Type(user)
		_ = structs.Name(user)
		_ = structs.IsStruct(user)
	}
}

func BenchmarkFieldOperations(b *testing.B) {
	user := User{ID: 1, Name: "Alice", Email: "alice@example.com"}

	b.ResetTimer()
	for b.Loop() {
		_ = structs.HasField(user, "Name")
		_, _ = structs.GetFieldValue(user, "Email")
	}
}

func BenchmarkValidation(b *testing.B) {
	user := User{
		ID:      1,
		Name:    "Alice",
		Email:   "alice@example.com",
		Age:     30,
		Active:  true,
		Profile: &Profile{Bio: "Engineer"},
		Tags:    []string{"admin"},
	}

	b.ResetTimer()
	for b.Loop() {
		_ = structs.NonZero(user)
	}
}

func BenchmarkCloning(b *testing.B) {
	user := User{
		ID:      1,
		Name:    "Alice",
		Email:   "alice@example.com",
		Profile: &Profile{Bio: "Engineer"},
		Tags:    []string{"admin"},
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = structs.Clone(user)
	}
}
