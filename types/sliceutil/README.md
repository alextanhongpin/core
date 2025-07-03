# SliceUtil

The `sliceutil` package provides comprehensive slice utilities that complement Go's standard library `slices` package. Built with generics, it offers functional programming patterns, mathematical operations, and advanced slice manipulations for modern Go development.

## Features

- **Functional Programming**: Map, Filter, Reduce, FlatMap operations
- **Conditional Operations**: All, Any, None predicates with element and index variants
- **Mathematical Functions**: Sum, Product, Min, Max, Average for numeric types
- **Query Operations**: Find, Contains, IndexOf with flexible predicates
- **Transformation Utilities**: Dedup, Reverse, Chunk, Flatten, Partition
- **Advanced Operations**: GroupBy, Zip/Unzip, Map with error handling
- **Type Safety**: Full generic support for any comparable or ordered types

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/alextanhongpin/core/types/sliceutil"
)

func main() {
    numbers := []int{1, 2, 3, 4, 5}
    
    // Transform data
    doubled := sliceutil.Map(numbers, func(n int) int { return n * 2 })
    fmt.Println("Doubled:", doubled) // [2, 4, 6, 8, 10]
    
    // Filter data
    evens := sliceutil.Filter(numbers, func(n int) bool { return n%2 == 0 })
    fmt.Println("Evens:", evens) // [2, 4]
    
    // Aggregate data
    sum := sliceutil.Sum(numbers)
    fmt.Println("Sum:", sum) // 15
}
```

## API Reference

### Conditional Operations

```go
// Check if all elements satisfy a condition
all := sliceutil.All([]int{2, 4, 6}, func(n int) bool { return n%2 == 0 })

// Check if any element satisfies a condition
any := sliceutil.Any([]int{1, 3, 4}, func(n int) bool { return n%2 == 0 })

// Check if no elements satisfy a condition
none := sliceutil.None([]int{1, 3, 5}, func(n int) bool { return n%2 == 0 })

// Index-based variants available: AllIndex, AnyIndex, NoneIndex
```

### Transformation Operations

```go
// Transform elements
doubled := sliceutil.Map([]int{1, 2, 3}, func(n int) int { return n * 2 })

// Transform with error handling
result, err := sliceutil.MapError([]string{"1", "2", "x"}, strconv.Atoi)

// Transform and flatten
tags := sliceutil.FlatMap(articles, func(a Article) []string { return a.Tags })

// Remove duplicates
unique := sliceutil.Dedup([]int{1, 2, 2, 3, 3, 3})
```

### Query Operations

```go
// Find first matching element
user, found := sliceutil.Find(users, func(u User) bool { 
    return u.Email == "john@example.com" 
})

// Find with index
user, index, found := sliceutil.FindIndex(users, func(u User) bool { 
    return u.Age > 30 
})

// Check membership
exists := sliceutil.Contains([]string{"a", "b", "c"}, "b")

// Get element indices
index := sliceutil.IndexOf([]string{"a", "b", "c"}, "b") // 1
```

### Mathematical Operations

```go
numbers := []int{1, 2, 3, 4, 5}

// Basic operations
sum := sliceutil.Sum(numbers)           // 15
product := sliceutil.Product(numbers)   // 120

// Statistical operations
min, ok := sliceutil.Min(numbers)       // 1, true
max, ok := sliceutil.Max(numbers)       // 5, true
avg, ok := sliceutil.Average(numbers)   // 3.0, true
```

### Advanced Operations

```go
// Group elements by key
grouped := sliceutil.GroupBy(users, func(u User) string { return u.Role })

// Split into chunks
batches := sliceutil.Chunk(orders, 100) // Process in batches of 100

// Partition by predicate
active, inactive := sliceutil.Partition(users, func(u User) bool { 
    return u.Active 
})

// Zip two slices
pairs := sliceutil.Zip([]string{"a", "b"}, []int{1, 2})
// Result: []struct{First string; Second int}{{a, 1}, {b, 2}}
```

### Slice Manipulation

```go
// Reverse elements
reversed := sliceutil.Reverse([]int{1, 2, 3}) // [3, 2, 1]

// Take/Drop elements
first3 := sliceutil.Take([]int{1, 2, 3, 4, 5}, 3)     // [1, 2, 3]
last2 := sliceutil.TakeLast([]int{1, 2, 3, 4, 5}, 2)  // [4, 5]
rest := sliceutil.Drop([]int{1, 2, 3, 4, 5}, 2)       // [3, 4, 5]

// Flatten nested slices
flat := sliceutil.Flatten([][]int{{1, 2}, {3, 4}, {5}}) // [1, 2, 3, 4, 5]
```

## Real-World Examples

### User Management System

```go
type User struct {
    ID     int
    Name   string
    Email  string
    Age    int
    Active bool
    Role   string
}

users := []User{
    {1, "Alice", "alice@example.com", 25, true, "admin"},
    {2, "Bob", "bob@example.com", 30, false, "user"},
    {3, "Charlie", "charlie@example.com", 35, true, "user"},
}

// Find active admin users
activeAdmins := sliceutil.Filter(users, func(u User) bool {
    return u.Active && u.Role == "admin"
})

// Get all email addresses
emails := sliceutil.Map(users, func(u User) string {
    return u.Email
})

// Group users by role
byRole := sliceutil.GroupBy(users, func(u User) string {
    return u.Role
})

// Check if all users are adults
allAdults := sliceutil.All(users, func(u User) bool {
    return u.Age >= 18
})
```

### Data Processing Pipeline

```go
// Process API response data
rawData := []string{"1", "2", "3", "invalid", "5"}

// Parse with error handling
numbers, err := sliceutil.MapError(rawData, strconv.Atoi)
if err != nil {
    // Handle parsing errors
}

// Filter and transform
evenSquares := sliceutil.Map(
    sliceutil.Filter(numbers, func(n int) bool { return n%2 == 0 }),
    func(n int) int { return n * n },
)

// Process in batches
batches := sliceutil.Chunk(evenSquares, 10)
for _, batch := range batches {
    // Process each batch
}
```

### Analytics and Reporting

```go
type Sale struct {
    Amount float64
    Date   time.Time
    Region string
}

sales := []Sale{
    {1500.50, time.Now(), "North"},
    {2300.75, time.Now(), "South"},
    {1800.25, time.Now(), "North"},
}

// Calculate total revenue
totalRevenue := sliceutil.Sum(sliceutil.Map(sales, func(s Sale) float64 {
    return s.Amount
}))

// Group sales by region
byRegion := sliceutil.GroupBy(sales, func(s Sale) string {
    return s.Region
})

// Find high-value sales
highValueSales := sliceutil.Filter(sales, func(s Sale) bool {
    return s.Amount > 2000
})

// Get sales statistics
amounts := sliceutil.Map(sales, func(s Sale) float64 { return s.Amount })
minSale, _ := sliceutil.Min(amounts)
maxSale, _ := sliceutil.Max(amounts)
avgSale, _ := sliceutil.Average(amounts)
```

### Content Management

```go
type Article struct {
    Title string
    Tags  []string
    Views int
}

articles := []Article{
    {"Go Basics", []string{"go", "programming"}, 1500},
    {"Advanced Go", []string{"go", "advanced"}, 800},
    {"Web Dev", []string{"web", "javascript"}, 2200},
}

// Get all unique tags
allTags := sliceutil.Dedup(sliceutil.FlatMap(articles, func(a Article) []string {
    return a.Tags
}))

// Find popular articles
popular := sliceutil.Filter(articles, func(a Article) bool {
    return a.Views > 1000
})

// Calculate total views
totalViews := sliceutil.Sum(sliceutil.Map(articles, func(a Article) int {
    return a.Views
}))
```

### Configuration Management

```go
type Config struct {
    Key     string
    Value   string
    Env     string
    Enabled bool
}

configs := []Config{
    {"db_host", "localhost", "dev", true},
    {"db_host", "prod.db.com", "prod", true},
    {"debug", "true", "dev", true},
    {"debug", "false", "prod", false},
}

// Group configurations by environment
byEnv := sliceutil.GroupBy(configs, func(c Config) string {
    return c.Env
})

// Get enabled configurations only
enabled := sliceutil.Filter(configs, func(c Config) bool {
    return c.Enabled
})

// Create key-value map for specific environment
devConfigs := byEnv["dev"]
configMap := make(map[string]string)
for _, config := range devConfigs {
    configMap[config.Key] = config.Value
}
```

## Performance Characteristics

- **Map/Filter/Reduce**: O(n) time complexity
- **Dedup**: O(n) average case with hash map
- **GroupBy**: O(n) average case
- **Chunk**: O(n) time, creates new slices
- **Memory Usage**: Most operations create new slices; use in-place operations when memory is critical

## Best Practices

1. **Choose the Right Function**: Use element-based functions (`Map`, `Filter`) unless you specifically need indices
2. **Error Handling**: Use `MapError` and similar functions when transformations can fail
3. **Memory Efficiency**: Consider memory usage for large slices; some operations create copies
4. **Type Safety**: Leverage Go's type system - functions are strongly typed
5. **Composition**: Chain operations for complex data transformations
6. **Benchmarking**: Profile performance-critical code, especially with large datasets

## Thread Safety

Functions in this package are not thread-safe. For concurrent access to shared slices, use appropriate synchronization mechanisms like `sync.RWMutex`.

## Migration from Index-Based Functions

If upgrading from index-based function signatures:

```go
// Old (index-based)
result := sliceutil.Map(slice, func(i int) string {
    return fmt.Sprintf("%d", slice[i])
})

// New (element-based)
result := sliceutil.Map(slice, func(item int) string {
    return fmt.Sprintf("%d", item)
})

// Use MapIndex if you need the index
result := sliceutil.MapIndex(slice, func(i int, item int) string {
    return fmt.Sprintf("%d:%d", i, item)
})
```
