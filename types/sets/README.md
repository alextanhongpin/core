# Sets

The `sets` package provides a generic, thread-safe Set data structure with comprehensive set operations. Built with Go generics, it supports any comparable type and offers both mathematical set operations and functional programming patterns.

## Features

- **Generic Implementation**: Works with any comparable type (`comparable` constraint)
- **Rich Set Operations**: Union, intersection, difference, symmetric difference
- **Set Relationships**: Subset, superset, disjoint, equal checks
- **Functional Helpers**: Map-like operations (Any, Every/All, Filter, ForEach)
- **Convenient Constructors**: Create sets from slices, other sets, or individual elements
- **Memory Efficient**: Uses Go's built-in map with empty struct values

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/alextanhongpin/core/types/sets"
)

func main() {
    // Create sets
    numbers := sets.From([]int{1, 2, 3, 4, 5})
    evens := sets.From([]int{2, 4, 6, 8})
    
    // Basic operations
    fmt.Println("Size:", numbers.Size())           // 5
    fmt.Println("Contains 3:", numbers.Has(3))    // true
    
    // Set operations
    intersection := numbers.Intersection(evens)
    fmt.Println("Intersection:", intersection.ToSlice()) // [2, 4]
    
    union := numbers.Union(evens)
    fmt.Println("Union size:", union.Size())             // 7
}
```

## API Reference

### Constructors

```go
// Create empty set
s := sets.New[int]()

// Create from slice
s := sets.From([]string{"a", "b", "c"})

// Create from individual elements  
s := sets.Of("x", "y", "z")
```

### Basic Operations

```go
// Add elements
s.Add("new")
s.AddAll([]string{"a", "b"})

// Remove elements
s.Remove("old")
s.RemoveAll([]string{"x", "y"})

// Check membership
exists := s.Has("item")

// Size and emptiness
size := s.Size()
empty := s.IsEmpty()

// Clear all elements
s.Clear()
```

### Set Operations

```go
a := sets.From([]int{1, 2, 3, 4})
b := sets.From([]int{3, 4, 5, 6})

// Union: {1, 2, 3, 4, 5, 6}
union := a.Union(b)

// Intersection: {3, 4}
intersection := a.Intersection(b)

// Difference: {1, 2}
diff := a.Difference(b)

// Symmetric difference: {1, 2, 5, 6}
symDiff := a.SymmetricDifference(b)
```

### Set Relationships

```go
a := sets.From([]int{1, 2})
b := sets.From([]int{1, 2, 3, 4})
c := sets.From([]int{5, 6})

// Subset checks
fmt.Println(a.IsSubset(b))    // true
fmt.Println(a.IsSuperset(b))  // false

// Disjoint sets (no common elements)
fmt.Println(a.IsDisjoint(c))  // true

// Set equality
fmt.Println(a.Equal(b))       // false
```

### Functional Operations

```go
numbers := sets.From([]int{1, 2, 3, 4, 5})

// Check conditions
hasEven := numbers.Any(func(n int) bool { return n%2 == 0 })
allPositive := numbers.Every(func(n int) bool { return n > 0 })

// Filter elements
evens := numbers.Filter(func(n int) bool { return n%2 == 0 })

// Iterate over elements
numbers.ForEach(func(n int) {
    fmt.Printf("Number: %d\n", n)
})
```

### Conversion and Cloning

```go
s := sets.From([]string{"a", "b", "c"})

// Convert to slice (order not guaranteed)
slice := s.ToSlice()

// Create independent copy
clone := s.Clone()
```

## Real-World Examples

### User Permission System

```go
// Define user roles and permissions
adminPerms := sets.From([]string{"read", "write", "delete", "admin"})
editorPerms := sets.From([]string{"read", "write"})
viewerPerms := sets.From([]string{"read"})

// Check if user can perform action
func canPerform(userPerms, requiredPerms *sets.Set[string]) bool {
    return userPerms.IsSuperset(requiredPerms)
}

// Usage
required := sets.From([]string{"read", "write"})
fmt.Println("Editor can edit:", canPerform(editorPerms, required)) // true
fmt.Println("Viewer can edit:", canPerform(viewerPerms, required)) // false
```

### Tag-Based Content Filtering

```go
// Content with tags
type Content struct {
    ID   string
    Tags *sets.Set[string]
}

articles := []Content{
    {"1", sets.From([]string{"tech", "programming", "go"})},
    {"2", sets.From([]string{"tech", "ai", "machine-learning"})},
    {"3", sets.From([]string{"lifestyle", "health"})},
}

// Filter content by tags
techFilter := sets.From([]string{"tech"})
filtered := make([]Content, 0)

for _, article := range articles {
    if !article.Tags.IsDisjoint(techFilter) {
        filtered = append(filtered, article)
    }
}
// Result: articles 1 and 2
```

### Feature Flag Management

```go
// Feature flags for different environments
prodFeatures := sets.From([]string{"auth", "payments", "analytics"})
stagingFeatures := sets.From([]string{"auth", "payments", "analytics", "debug"})
devFeatures := sets.From([]string{"auth", "payments", "analytics", "debug", "mock-data"})

// Check environment-specific features
func isFeatureEnabled(env string, feature string) bool {
    switch env {
    case "prod":
        return prodFeatures.Has(feature)
    case "staging":
        return stagingFeatures.Has(feature)
    case "dev":
        return devFeatures.Has(feature)
    default:
        return false
    }
}

// Find features only in development
devOnlyFeatures := devFeatures.Difference(prodFeatures)
fmt.Println("Dev-only features:", devOnlyFeatures.ToSlice()) // ["debug", "mock-data"]
```

### Data Deduplication

```go
// Combine data from multiple sources while avoiding duplicates
source1 := []string{"apple", "banana", "cherry"}
source2 := []string{"banana", "date", "elderberry"}
source3 := []string{"cherry", "fig", "grape"}

// Combine all unique items
combined := sets.New[string]()
combined.AddAll(source1)
combined.AddAll(source2)
combined.AddAll(source3)

uniqueItems := combined.ToSlice()
fmt.Printf("Unique items: %v\n", uniqueItems)
// Result: All unique fruits from all sources
```

## Performance Characteristics

- **Add/Remove/Contains**: O(1) average case
- **Set Operations**: O(n) where n is the size of the larger set
- **Memory**: O(n) where n is the number of unique elements
- **Thread Safety**: Not thread-safe by default (use external synchronization if needed)

## Thread Safety

The Set is not thread-safe. For concurrent access, wrap operations with appropriate synchronization:

```go
import "sync"

type SafeSet[T comparable] struct {
    set *sets.Set[T]
    mu  sync.RWMutex
}

func (s *SafeSet[T]) Add(item T) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.set.Add(item)
}

func (s *SafeSet[T]) Has(item T) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.set.Has(item)
}
```

## Best Practices

1. **Use appropriate constructors**: `From()` for slices, `Of()` for individual elements
2. **Check emptiness**: Use `IsEmpty()` rather than `Size() == 0`
3. **Immutable operations**: Set operations return new sets, leaving originals unchanged
4. **Memory management**: Use `Clear()` to reuse sets rather than creating new ones
5. **Type safety**: Leverage Go's type system - all elements must be the same comparable type
