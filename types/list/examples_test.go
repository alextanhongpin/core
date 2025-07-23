package list_test

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/types/list"
	"github.com/stretchr/testify/assert"
)

// Example: User Management System
type User struct {
	ID     int
	Name   string
	Email  string
	Age    int
	Active bool
	Role   string
}

func ExampleAll_userValidation() {
	users := []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com", Age: 25, Active: true},
		{ID: 2, Name: "Bob", Email: "bob@example.com", Age: 30, Active: true},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com", Age: 35, Active: true},
	}

	// Check if all users are active
	allActive := list.All(users, func(u User) bool {
		return u.Active
	})

	fmt.Printf("All users active: %v\n", allActive)
	// Output: All users active: true
}

func ExampleAny_roleCheck() {
	users := []User{
		{ID: 1, Name: "Alice", Role: "user"},
		{ID: 2, Name: "Bob", Role: "admin"},
		{ID: 3, Name: "Charlie", Role: "user"},
	}

	// Check if any user is an admin
	hasAdmin := list.Any(users, func(u User) bool {
		return u.Role == "admin"
	})

	fmt.Printf("Has admin user: %v\n", hasAdmin)
	// Output: Has admin user: true
}

func ExampleFilter_activeUsers() {
	users := []User{
		{ID: 1, Name: "Alice", Active: true, Age: 25},
		{ID: 2, Name: "Bob", Active: false, Age: 30},
		{ID: 3, Name: "Charlie", Active: true, Age: 35},
	}

	// Filter active users over 30
	activeAdults := list.Filter(users, func(u User) bool {
		return u.Active && u.Age >= 30
	})

	fmt.Printf("Active adults: %d users\n", len(activeAdults))
	// Output: Active adults: 1 users
}

func ExampleMap_emailExtraction() {
	users := []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com"},
	}

	// Extract emails
	emails := list.Map(users, func(u User) string {
		return u.Email
	})

	fmt.Printf("Emails: %v\n", emails)
	// Output: Emails: [alice@example.com bob@example.com charlie@example.com]
}

func ExampleMapError_userValidation() {
	userInputs := []string{"alice@example.com", "invalid-email", "bob@example.com"}

	// Validate and transform emails
	validEmails, err := list.MapError(userInputs, func(email string) (string, error) {
		if !strings.Contains(email, "@") {
			return "", errors.New("invalid email format")
		}
		return strings.ToLower(email), nil
	})

	if err != nil {
		fmt.Printf("Validation error: %v\n", err)
	} else {
		fmt.Printf("Valid emails: %v\n", validEmails)
	}
	// Output: Validation error: invalid email format
}

func ExampleFind_userLookup() {
	users := []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com"},
	}

	// Find user by email
	user, found := list.Find(users, func(u User) bool {
		return u.Email == "bob@example.com"
	})

	if found {
		fmt.Printf("Found user: %s\n", user.Name)
	}
	// Output: Found user: Bob
}

func ExampleGroupBy_usersByRole() {
	users := []User{
		{ID: 1, Name: "Alice", Role: "admin"},
		{ID: 2, Name: "Bob", Role: "user"},
		{ID: 3, Name: "Charlie", Role: "admin"},
		{ID: 4, Name: "David", Role: "user"},
	}

	// Group users by role
	grouped := list.GroupBy(users, func(u User) string {
		return u.Role
	})

	for role, roleUsers := range grouped {
		fmt.Printf("%s: %d users\n", role, len(roleUsers))
	}
	// Output: admin: 2 users
	// user: 2 users
}

// Example: Data Processing Pipeline
func ExampleChunk_batchProcessing() {
	orders := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// Process orders in batches of 3
	batches := list.Chunk(orders, 3)

	for i, batch := range batches {
		fmt.Printf("Batch %d: %v\n", i+1, batch)
	}
	// Output: Batch 1: [1 2 3]
	// Batch 2: [4 5 6]
	// Batch 3: [7 8 9]
	// Batch 4: [10]
}

func ExampleFlatMap_tagExpansion() {
	articles := []struct {
		Title string
		Tags  []string
	}{
		{"Go Basics", []string{"go", "programming", "tutorial"}},
		{"Advanced Go", []string{"go", "advanced", "programming"}},
		{"Web Development", []string{"web", "javascript", "frontend"}},
	}

	// Extract all unique tags
	allTags := list.FlatMap(articles, func(article struct {
		Title string
		Tags  []string
	}) []string {
		return article.Tags
	})

	uniqueTags := list.Dedup(allTags)
	fmt.Printf("Unique tags: %v\n", uniqueTags)
	// Output: Unique tags: [go programming tutorial advanced web javascript frontend]
}

func ExampleReduce_statistics() {
	numbers := []int{1, 2, 3, 4, 5}

	// Calculate sum using reduce
	sum := list.Reduce(numbers, 0, func(acc, n int) int {
		return acc + n
	})

	// Calculate product using reduce
	product := list.Reduce(numbers, 1, func(acc, n int) int {
		return acc * n
	})

	fmt.Printf("Sum: %d, Product: %d\n", sum, product)
	// Output: Sum: 15, Product: 120
}

// Example: Analytics and Metrics
func ExampleSum_salesAnalytics() {
	dailySales := []float64{1500.50, 2300.75, 1800.25, 2100.00, 1650.80}

	totalSales := list.Sum(dailySales)
	average := list.Average(dailySales)

	fmt.Printf("Total sales: $%.2f\n", totalSales)
	fmt.Printf("Average daily sales: $%.2f\n", average)
	// Output: Total sales: $9352.30
	// Average daily sales: $1870.46
}

func ExamplePartition_orderProcessing() {
	orders := []struct {
		ID     int
		Amount float64
		Status string
	}{
		{1, 150.00, "completed"},
		{2, 75.50, "pending"},
		{3, 200.00, "completed"},
		{4, 120.25, "cancelled"},
		{5, 300.00, "completed"},
	}

	// Partition orders by completion status
	completed, pending := list.Partition(orders, func(order struct {
		ID     int
		Amount float64
		Status string
	}) bool {
		return order.Status == "completed"
	})

	fmt.Printf("Completed orders: %d\n", len(completed))
	fmt.Printf("Non-completed orders: %d\n", len(pending))
	// Output: Completed orders: 3
	// Non-completed orders: 2
}

// Example: Text Processing
func ExampleDedup_wordFrequency() {
	words := []string{"apple", "banana", "apple", "cherry", "banana", "date", "apple"}

	// Remove duplicates while preserving order
	uniqueWords := list.Dedup(words)

	fmt.Printf("Original: %v\n", words)
	fmt.Printf("Unique: %v\n", uniqueWords)
	// Output: Original: [apple banana apple cherry banana date apple]
	// Unique: [apple banana cherry date]
}

func ExampleDedupFunc_caseInsensitive() {
	words := []string{"Apple", "BANANA", "apple", "Cherry", "banana", "DATE"}

	// Remove duplicates case-insensitively
	uniqueWords := list.DedupFunc(words, func(word string) string {
		return strings.ToLower(word)
	})

	fmt.Printf("Case-insensitive unique: %v\n", uniqueWords)
	// Output: Case-insensitive unique: [Apple BANANA Cherry DATE]
}

// Example: Mathematical Operations
func ExampleMin_temperatureAnalysis() {
	temperatures := []float64{23.5, 18.2, 31.8, 15.9, 28.7, 22.1}

	minTemp := slices.Min(temperatures)
	maxTemp := slices.Max(temperatures)
	avgTemp := list.Average(temperatures)

	fmt.Printf("Temperature Analysis:\n")
	fmt.Printf("Min: %.1f°C\n", minTemp)
	fmt.Printf("Max: %.1f°C\n", maxTemp)
	fmt.Printf("Average: %.1f°C\n", avgTemp)
	// Output: Temperature Analysis:
	// Min: 15.9°C
	// Max: 31.8°C
	// Average: 23.4°C
}

// Example: Data Transformation Pipeline
func ExampleFilter_complexPipeline() {
	// Simulate API response data
	rawData := []string{"1", "2", "3", "invalid", "5", "6", "7", "8", "9", "10"}

	// Pipeline: parse -> filter -> transform -> chunk
	// Step 1: Parse strings to integers, handle errors
	numbers := list.Map(rawData, func(s string) int {
		n, err := strconv.Atoi(s)
		if err != nil {
			return -1
		}
		return n
	})

	// Step 2: Filter even numbers
	evenNumbers := list.Filter(numbers, func(n int) bool {
		return n%2 == 0
	})

	// Step 3: Square the numbers
	squared := list.Map(evenNumbers, func(n int) int {
		return n * n
	})

	// Step 4: Process in batches
	batches := list.Chunk(squared, 2)

	for i, batch := range batches {
		fmt.Printf("Batch %d: %v\n", i+1, batch)
	}
	// Output: Batch 1: [4 36]
	// Batch 2: [64 100]
}

// Benchmarks
func BenchmarkMap(b *testing.B) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = list.Map(data, func(n int) int {
			return n * 2
		})
	}
}

func BenchmarkFilter(b *testing.B) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = list.Filter(data, func(n int) bool {
			return n%2 == 0
		})
	}
}

func BenchmarkDedup(b *testing.B) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i % 100 // Create duplicates
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = list.Dedup(data)
	}
}

// Unit Tests
func TestSliceUtilFunctions(t *testing.T) {
	t.Run("All", func(t *testing.T) {
		assert := assert.New(t)

		// All positive numbers
		assert.True(list.All([]int{1, 2, 3, 4}, func(n int) bool { return n > 0 }))

		// Not all even numbers
		assert.False(list.All([]int{1, 2, 3, 4}, func(n int) bool { return n%2 == 0 }))

		// Empty slice
		assert.False(list.All([]int{}, func(n int) bool { return n > 0 }))
	})

	t.Run("Any", func(t *testing.T) {
		assert := assert.New(t)

		// Any even number
		assert.True(list.Any([]int{1, 2, 3, 4}, func(n int) bool { return n%2 == 0 }))

		// No negative numbers
		assert.False(list.Any([]int{1, 2, 3, 4}, func(n int) bool { return n < 0 }))

		// Empty slice
		assert.False(list.Any([]int{}, func(n int) bool { return n > 0 }))
	})

	t.Run("Sum", func(t *testing.T) {
		assert := assert.New(t)

		assert.Equal(15, list.Sum([]int{1, 2, 3, 4, 5}))
		assert.Equal(0, list.Sum([]int{}))
		assert.Equal(15.5, list.Sum([]float64{1.5, 2.5, 3.5, 4.5, 3.5}))
	})

	t.Run("Map", func(t *testing.T) {
		assert := assert.New(t)

		doubled := list.Map([]int{1, 2, 3}, func(n int) int { return n * 2 })
		assert.Equal([]int{2, 4, 6}, doubled)

		strings := list.Map([]int{1, 2, 3}, func(n int) string { return fmt.Sprintf("num_%d", n) })
		assert.Equal([]string{"num_1", "num_2", "num_3"}, strings)
	})

	t.Run("Filter", func(t *testing.T) {
		assert := assert.New(t)

		evens := list.Filter([]int{1, 2, 3, 4, 5, 6}, func(n int) bool { return n%2 == 0 })
		assert.Equal([]int{2, 4, 6}, evens)

		empty := list.Filter([]int{1, 3, 5}, func(n int) bool { return n%2 == 0 })
		assert.Equal([]int{}, empty)
	})

	t.Run("Dedup", func(t *testing.T) {
		assert := assert.New(t)

		unique := list.Dedup([]int{1, 2, 2, 3, 3, 3, 4})
		assert.ElementsMatch([]int{1, 2, 3, 4}, unique)

		// Order preservation test
		uniquePreserved := list.Dedup([]string{"a", "b", "a", "c", "b"})
		expected := []string{"a", "b", "c"}
		assert.Equal(expected, uniquePreserved)
	})

	t.Run("Chunk", func(t *testing.T) {
		assert := assert.New(t)

		chunks := list.Chunk([]int{1, 2, 3, 4, 5, 6, 7}, 3)
		expected := [][]int{{1, 2, 3}, {4, 5, 6}, {7}}
		assert.Equal(expected, chunks)

		// Empty slice
		emptyChunks := list.Chunk([]int{}, 3)
		assert.Equal([][]int{}, emptyChunks)

		// Invalid chunk size
		nilChunks := list.Chunk([]int{1, 2, 3}, 0)
		assert.Nil(nilChunks)
	})

	t.Run("Partition", func(t *testing.T) {
		assert := assert.New(t)

		evens, odds := list.Partition([]int{1, 2, 3, 4, 5, 6}, func(n int) bool { return n%2 == 0 })
		assert.Equal([]int{2, 4, 6}, evens)
		assert.Equal([]int{1, 3, 5}, odds)
	})
}
