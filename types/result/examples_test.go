package result_test

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alextanhongpin/core/types/result"
)

// Example: HTTP API with Result pattern
func ExampleResult_httpAPI() {
	// Simulate API client functions that return Results
	fetchUser := func(id int) *result.Result[User] {
		if id <= 0 {
			return result.Err[User](errors.New("invalid user ID"))
		}
		if id == 404 {
			return result.Err[User](errors.New("user not found"))
		}
		return result.OK(User{ID: id, Name: fmt.Sprintf("User%d", id), Email: fmt.Sprintf("user%d@example.com", id)})
	}

	fetchProfile := func(userID int) *result.Result[Profile] {
		if userID == 999 {
			return result.Err[Profile](errors.New("profile service unavailable"))
		}
		return result.OK(Profile{UserID: userID, Bio: "A sample bio", Avatar: "avatar.jpg"})
	}

	// Get user and then fetch profile separately
	userResult := fetchUser(123)
	if user, err := userResult.Unwrap(); err != nil {
		fmt.Printf("Error fetching user: %v\n", err)
	} else {
		profileResult := fetchProfile(user.ID)
		if profile, err := profileResult.Unwrap(); err != nil {
			fmt.Printf("Error fetching profile: %v\n", err)
		} else {
			fmt.Printf("Profile for user %d: %s\n", profile.UserID, profile.Bio)
		}
	}

	// Handle error case
	errorResult := fetchUser(404)
	user := errorResult.UnwrapOr(User{ID: 0, Name: "Default", Email: "default@example.com"})
	fmt.Printf("Default user: %s\n", user.Name)

	// Output:
	// Profile for user 123: A sample bio
	// Default user: Default
}

// Example: Concurrent operations with Results
func ExampleAll_concurrentOperations() {
	fmt.Println("Fetching data from multiple services...")

	// Simulate concurrent API calls
	fetchService1 := func() *result.Result[string] {
		time.Sleep(10 * time.Millisecond) // Simulate network delay
		return result.OK("Service1 data")
	}

	fetchService2 := func() *result.Result[string] {
		time.Sleep(15 * time.Millisecond)
		return result.OK("Service2 data")
	}

	fetchService3 := func() *result.Result[string] {
		time.Sleep(5 * time.Millisecond)
		return result.OK("Service3 data")
	}

	// Collect all results
	results := []*result.Result[string]{
		fetchService1(),
		fetchService2(),
		fetchService3(),
	}

	// Wait for all to complete and combine
	if data, err := result.All(results...); err != nil {
		fmt.Printf("Error collecting data: %v\n", err)
	} else {
		fmt.Printf("Collected data: %v\n", data)
	}

	// Output:
	// Fetching data from multiple services...
	// Collected data: [Service1 data Service2 data Service3 data]
}

// Example: Fallback pattern with Any
func ExampleAny_fallbackPattern() {
	fmt.Println("Trying multiple data sources...")

	// Simulate multiple data sources with different reliability
	primaryDB := func() *result.Result[string] {
		return result.Err[string](errors.New("primary database down"))
	}

	secondaryDB := func() *result.Result[string] {
		return result.Err[string](errors.New("secondary database timeout"))
	}

	cache := func() *result.Result[string] {
		return result.OK("cached data")
	}

	defaultData := func() *result.Result[string] {
		return result.OK("default fallback data")
	}

	// Try sources in order of preference
	data, err := result.Any(
		primaryDB(),
		secondaryDB(),
		cache(),
		defaultData(),
	)

	if err != nil {
		fmt.Printf("All sources failed: %v\n", err)
	} else {
		fmt.Printf("Retrieved data: %s\n", data)
	}

	// Output:
	// Trying multiple data sources...
	// Retrieved data: cached data
}

// Example: Data pipeline with transformations
func ExampleResult_dataPipeline() {
	fmt.Println("Processing data pipeline...")

	// Pipeline stages
	parseInput := func(input string) *result.Result[int] {
		if input == "" {
			return result.Err[int](errors.New("empty input"))
		}
		val, err := strconv.Atoi(input)
		if err != nil {
			return result.Err[int](fmt.Errorf("parse error: %w", err))
		}
		return result.OK(val)
	}

	validate := func(val int) *result.Result[int] {
		if val < 0 {
			return result.Err[int](errors.New("negative values not allowed"))
		}
		if val > 1000 {
			return result.Err[int](errors.New("value too large"))
		}
		return result.OK(val)
	}

	transform := func(val int) *result.Result[float64] {
		// Apply some business logic transformation
		transformed := float64(val) * 1.1 // 10% markup
		return result.OK(transformed)
	}

	// Process multiple inputs
	inputs := []string{"100", "invalid", "50", "-10", "2000"}

	for i, input := range inputs {
		parseResult := parseInput(input)
		validatedResult := parseResult.FlatMap(validate)

		if val, err := validatedResult.Unwrap(); err != nil {
			fmt.Printf("Input %d (%s): Error - %v\n", i+1, input, err)
		} else {
			transformResult := transform(val)
			if value, err := transformResult.Unwrap(); err != nil {
				fmt.Printf("Input %d (%s): Transform Error - %v\n", i+1, input, err)
			} else {
				fmt.Printf("Input %d (%s): Success - %.2f\n", i+1, input, value)
			}
		}
	}

	// Output:
	// Processing data pipeline...
	// Input 1 (100): Success - 110.00
	// Input 2 (invalid): Error - parse error: strconv.Atoi: parsing "invalid": invalid syntax
	// Input 3 (50): Success - 55.00
	// Input 4 (-10): Error - negative values not allowed
	// Input 5 (2000): Error - value too large
}

// Example: Configuration loading with multiple sources
func ExampleResult_configurationLoading() {
	fmt.Println("Loading configuration...")

	// Configuration sources
	loadFromEnv := func(key string) *result.Result[string] {
		// Simulate environment variable lookup
		envVars := map[string]string{
			"APP_NAME": "MyApp",
			"DEBUG":    "true",
		}
		if val, exists := envVars[key]; exists {
			return result.OK(val)
		}
		return result.Err[string](fmt.Errorf("env var %s not found", key))
	}

	loadFromFile := func(key string) *result.Result[string] {
		// Simulate config file lookup
		configFile := map[string]string{
			"database_url": "postgres://localhost/mydb",
			"port":         "8080",
		}
		if val, exists := configFile[key]; exists {
			return result.OK(val)
		}
		return result.Err[string](fmt.Errorf("config key %s not found in file", key))
	}

	getDefault := func(key string, defaultVal string) *result.Result[string] {
		return result.OK(defaultVal)
	}

	// Load configuration with fallbacks
	configKeys := map[string]string{
		"APP_NAME":     "DefaultApp",
		"database_url": "sqlite://memory",
		"port":         "3000",
		"timeout":      "30s",
	}

	config := make(map[string]string)
	for key, defaultVal := range configKeys {
		// Try multiple sources with fallback
		value, _ := result.Any(
			loadFromEnv(key),
			loadFromFile(key),
			getDefault(key, defaultVal),
		)
		config[key] = value
	}

	fmt.Printf("Loaded configuration: %+v\n", config)

	// Output:
	// Loading configuration...
	// Loaded configuration: map[APP_NAME:MyApp database_url:postgres://localhost/mydb port:8080 timeout:30s]
}

// Example: Batch processing with error handling
func ExampleFilter_batchProcessing() {
	fmt.Println("Processing batch of records...")

	// Simulate record processing
	processRecord := func(id int) *result.Result[ProcessedRecord] {
		// Simulate various failure conditions
		switch {
		case id%7 == 0: // Every 7th record fails
			return result.Err[ProcessedRecord](fmt.Errorf("processing failed for ID %d", id))
		case id%13 == 0: // Every 13th record has validation error
			return result.Err[ProcessedRecord](fmt.Errorf("validation error for ID %d", id))
		default:
			return result.OK(ProcessedRecord{
				ID:        id,
				Status:    "processed",
				Timestamp: time.Now().Unix(),
			})
		}
	}

	// Process batch of 20 records
	var results []*result.Result[ProcessedRecord]
	for i := 1; i <= 20; i++ {
		results = append(results, processRecord(i))
	}

	// Separate successful and failed results
	successful := result.Filter(results...)
	successfulRecords, errors := result.Partition(results...)

	fmt.Printf("Batch processing complete:\n")
	fmt.Printf("- Successful: %d records\n", len(successful))
	fmt.Printf("- Failed: %d records\n", len(errors))

	// Show first few successful records
	for i, record := range successfulRecords[:min(3, len(successfulRecords))] {
		fmt.Printf("  Success %d: ID=%d, Status=%s\n", i+1, record.ID, record.Status)
	}

	// Show first few errors
	for i, err := range errors[:min(2, len(errors))] {
		fmt.Printf("  Error %d: %v\n", i+1, err)
	}

	// Output:
	// Processing batch of records...
	// Batch processing complete:
	// - Successful: 16 records
	// - Failed: 4 records
	//   Success 1: ID=1, Status=processed
	//   Success 2: ID=2, Status=processed
	//   Success 3: ID=3, Status=processed
	//   Error 1: processing failed for ID 7
	//   Error 2: validation error for ID 13
}

// Example: HTTP client with Result pattern
func ExampleFrom_httpClient() {
	fmt.Println("Making HTTP requests with Result pattern...")

	// HTTP client wrapper
	makeRequest := func(url string) *result.Result[APIResponse] {
		return result.From(func() (APIResponse, error) {
			// Simulate HTTP request
			switch {
			case strings.Contains(url, "invalid"):
				return APIResponse{}, errors.New("invalid URL")
			case strings.Contains(url, "timeout"):
				return APIResponse{}, errors.New("request timeout")
			case strings.Contains(url, "500"):
				return APIResponse{}, errors.New("server error")
			default:
				return APIResponse{
					Status: 200,
					Data:   fmt.Sprintf("Response from %s", url),
				}, nil
			}
		})
	}

	// Test different URLs
	urls := []string{
		"https://api.example.com/users",
		"https://api.example.com/invalid",
		"https://api.example.com/timeout",
		"https://api.example.com/posts",
	}

	for _, url := range urls {
		response := makeRequest(url).
			Map(func(resp APIResponse) APIResponse {
				// Transform successful responses
				resp.Data = "Processed: " + resp.Data
				return resp
			}).
			MapError(func(err error) error {
				// Enhance error information
				return fmt.Errorf("API request failed for %s: %w", url, err)
			})

		if resp, err := response.Unwrap(); err != nil {
			fmt.Printf("❌ %v\n", err)
		} else {
			fmt.Printf("✅ %s (Status: %d)\n", resp.Data, resp.Status)
		}
	}

	// Output:
	// Making HTTP requests with Result pattern...
	// ✅ Processed: Response from https://api.example.com/users (Status: 200)
	// ❌ API request failed for https://api.example.com/invalid: invalid URL
	// ❌ API request failed for https://api.example.com/timeout: request timeout
	// ✅ Processed: Response from https://api.example.com/posts (Status: 200)
}

// Example: Database operations with transactions
func ExampleResult_databaseTransactions() {
	fmt.Println("Database transaction simulation...")

	// Simulate database operations
	type Transaction struct {
		ID     string
		Amount float64
	}

	validateTransaction := func(tx Transaction) *result.Result[Transaction] {
		if tx.Amount <= 0 {
			return result.Err[Transaction](errors.New("invalid amount"))
		}
		if tx.Amount > 10000 {
			return result.Err[Transaction](errors.New("amount exceeds limit"))
		}
		return result.OK(tx)
	}

	checkBalance := func(tx Transaction) *result.Result[Transaction] {
		// Simulate balance check
		if tx.ID == "insufficient" {
			return result.Err[Transaction](errors.New("insufficient funds"))
		}
		return result.OK(tx)
	}

	recordTransaction := func(tx Transaction) *result.Result[string] {
		// Simulate database write
		if tx.ID == "db_error" {
			return result.Err[string](errors.New("database write failed"))
		}
		return result.OK(fmt.Sprintf("TXN_%s_%d", tx.ID, time.Now().Unix()))
	}

	// Process transactions
	transactions := []Transaction{
		{ID: "user1", Amount: 100.50},
		{ID: "user2", Amount: -50.00},       // Invalid amount
		{ID: "insufficient", Amount: 75.00}, // Insufficient funds
		{ID: "user3", Amount: 15000.00},     // Exceeds limit
		{ID: "user4", Amount: 200.00},
	}

	for i, tx := range transactions {
		validatedTx := validateTransaction(tx)
		checkedTx := validatedTx.FlatMap(checkBalance)

		if finalTx, err := checkedTx.Unwrap(); err != nil {
			fmt.Printf("Transaction %d failed: %v\n", i+1, err)
		} else {
			txnResult := recordTransaction(finalTx)
			if txnID, err := txnResult.Unwrap(); err != nil {
				fmt.Printf("Transaction %d failed: %v\n", i+1, err)
			} else {
				fmt.Printf("Transaction %d successful: %s\n", i+1, txnID)
			}
		}
	}

	// Output:
	// Database transaction simulation...
	// Transaction 1 successful: TXN_user1_1640995200
	// Transaction 2 failed: invalid amount
	// Transaction 3 failed: insufficient funds
	// Transaction 4 failed: amount exceeds limit
	// Transaction 5 successful: TXN_user4_1640995200
}

// Supporting types for examples
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Profile struct {
	UserID int    `json:"user_id"`
	Bio    string `json:"bio"`
	Avatar string `json:"avatar"`
}

type ProcessedRecord struct {
	ID        int    `json:"id"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

type APIResponse struct {
	Status int    `json:"status"`
	Data   string `json:"data"`
}

// Test Result operations
func TestResultOperations(t *testing.T) {
	t.Run("OK result", func(t *testing.T) {
		r := result.OK(42)

		if !r.IsOK() {
			t.Error("Expected result to be OK")
		}

		if r.IsErr() {
			t.Error("Expected result not to be error")
		}

		value, err := r.Unwrap()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if value != 42 {
			t.Errorf("Expected value 42, got %d", value)
		}
	})

	t.Run("Error result", func(t *testing.T) {
		expectedErr := errors.New("test error")
		r := result.Err[int](expectedErr)

		if r.IsOK() {
			t.Error("Expected result to be error")
		}

		if !r.IsErr() {
			t.Error("Expected result to be error")
		}

		_, err := r.Unwrap()
		if err == nil {
			t.Error("Expected error, got nil")
		}

		if err.Error() != expectedErr.Error() {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("UnwrapOr", func(t *testing.T) {
		okResult := result.OK(42)
		errResult := result.Err[int](errors.New("error"))

		if value := okResult.UnwrapOr(0); value != 42 {
			t.Errorf("Expected 42, got %d", value)
		}

		if value := errResult.UnwrapOr(100); value != 100 {
			t.Errorf("Expected 100, got %d", value)
		}
	})

	t.Run("Map", func(t *testing.T) {
		r := result.OK(5)
		mapped := r.Map(func(x int) int { return x * 2 })

		value, err := mapped.Unwrap()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if value != 10 {
			t.Errorf("Expected 10, got %d", value)
		}
	})

	t.Run("FlatMap", func(t *testing.T) {
		r := result.OK(5)
		flatMapped := r.FlatMap(func(x int) *result.Result[int] {
			if x > 0 {
				return result.OK(x * 2)
			}
			return result.Err[int](errors.New("negative value"))
		})

		value, err := flatMapped.Unwrap()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if value != 10 {
			t.Errorf("Expected 10, got %d", value)
		}
	})
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func init() {
	// Suppress output in tests unless running examples
	log.SetOutput(io.Discard)
}
