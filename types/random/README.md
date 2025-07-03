# Random Package

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/types/random.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/types/random)

The `random` package provides cryptographically secure random value generation utilities for Go applications. It leverages Go's `math/rand/v2` package which uses ChaCha8 as its PRNG for security and performance.

## Features

- **Duration Generation**: Random time durations for delays, timeouts, and intervals
- **Numeric Generation**: Random integers and floats with flexible ranges
- **Boolean Generation**: Random boolean values with configurable probabilities
- **Collection Operations**: Random selection, sampling, and shuffling
- **String Generation**: Random strings with customizable character sets
- **Type Safety**: Generic functions that work with any comparable type
- **Performance**: Efficient implementations suitable for high-frequency use

## Installation

```bash
go get github.com/alextanhongpin/core/types/random
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/types/random"
)

func main() {
    // Random duration (0 to 5 seconds)
    delay := random.Duration(5 * time.Second)
    fmt.Printf("Random delay: %v\n", delay)
    
    // Random integer between 1 and 100
    score := random.IntBetween(1, 101)
    fmt.Printf("Random score: %d\n", score)
    
    // Random choice from slice
    colors := []string{"red", "green", "blue", "yellow"}
    color := random.Choice(colors)
    fmt.Printf("Random color: %s\n", color)
    
    // Random session ID
    sessionID := random.AlphaNumeric(32)
    fmt.Printf("Session ID: %s\n", sessionID)
}
```

## Core Functions

### Duration Generation

```go
// Random duration up to maximum
delay := random.Duration(10 * time.Second)

// Random duration between min and max
backoff := random.DurationBetween(1*time.Second, 5*time.Second)
```

**Use Cases:**
- Retry backoff with jitter
- Load testing delays
- Random timeouts
- Simulation intervals

### Numeric Generation

```go
// Random integers
dice := random.IntBetween(1, 7)        // 1-6 (dice roll)
percentage := random.Int(101)          // 0-100

// Random floats
temperature := random.FloatBetween(20.0, 30.0)  // 20.0-30.0°C
factor := random.Float(1.0)                     // 0.0-1.0
```

**Use Cases:**
- Game mechanics (damage, dice rolls)
- Statistical simulations
- Random coefficients
- Test data generation

### Boolean Generation

```go
// 50/50 chance
coinFlip := random.Bool()

// Custom probability (30% chance of true)
feature := random.BoolWithProbability(0.3)
```

**Use Cases:**
- A/B testing
- Feature flags
- Random failures (chaos engineering)
- Conditional logic

### Collection Operations

```go
names := []string{"Alice", "Bob", "Charlie", "Diana"}

// Random selection
winner := random.Choice(names)

// Multiple selections with replacement
participants := random.Choices(names, 3)

// Sampling without replacement
finalists := random.Sample(names, 2)

// Shuffle in place
random.Shuffle(names)
```

**Use Cases:**
- Content recommendation
- Playlist shuffling
- Random sampling
- Tournament brackets

### String Generation

```go
// Alphanumeric string
userID := random.AlphaNumeric(16)

// Custom character set
pin := random.String(4, "0123456789")

// Hexadecimal
color := "#" + random.Hex(6)
```

**Use Cases:**
- Session IDs
- Temporary passwords
- API keys
- Test data

## Real-World Examples

### Load Testing with Jitter

Prevent thundering herd problems by adding random delays:

```go
func loadTest() {
    for i := 0; i < 100; i++ {
        // Random delay to spread load
        jitter := random.Duration(2 * time.Second)
        time.Sleep(jitter)
        
        // Make request
        makeHTTPRequest()
    }
}
```

### Exponential Backoff with Jitter

Implement robust retry logic:

```go
func retryWithBackoff(operation func() error) error {
    baseDelay := 100 * time.Millisecond
    
    for attempt := 0; attempt < 5; attempt++ {
        if err := operation(); err == nil {
            return nil
        }
        
        // Exponential backoff with jitter
        backoff := time.Duration(1<<attempt) * baseDelay
        jitter := backoff / 4
        delay := random.DurationBetween(backoff-jitter, backoff+jitter)
        
        time.Sleep(delay)
    }
    
    return errors.New("max retries exceeded")
}
```

### A/B Testing

Implement feature rollouts:

```go
func shouldShowNewFeature(userID string) bool {
    // 30% of users get the new feature
    return random.BoolWithProbability(0.3)
}

func handleRequest(userID string) {
    if shouldShowNewFeature(userID) {
        serveNewFeature()
    } else {
        serveOldFeature()
    }
}
```

### Game Mechanics

Create engaging randomness:

```go
type Character struct {
    Level  int
    Weapon struct {
        BaseDamage int
    }
}

func (c *Character) Attack() int {
    // Random damage ±20% of base
    base := c.Weapon.BaseDamage
    min := base * 80 / 100
    max := base * 120 / 100
    
    damage := random.IntBetween(min, max+1)
    
    // Critical hit (5% chance for double damage)
    if random.BoolWithProbability(0.05) {
        damage *= 2
    }
    
    return damage
}
```

### Content Recommendation

Build recommendation systems:

```go
func recommendContent(userPreferences []string, allContent []string) []string {
    recommendations := make([]string, 0, 5)
    
    for i := 0; i < 5; i++ {
        // 70% chance to use preferences, 30% for discovery
        if random.BoolWithProbability(0.7) && len(userPreferences) > 0 {
            content := random.Choice(userPreferences)
            recommendations = append(recommendations, content)
        } else {
            content := random.Choice(allContent)
            recommendations = append(recommendations, content)
        }
    }
    
    return recommendations
}
```

### Chaos Engineering

Test system resilience:

```go
func chaosMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 2% chance of random failure
        if random.BoolWithProbability(0.02) {
            http.Error(w, "Random chaos failure", http.StatusInternalServerError)
            return
        }
        
        // Random delay (0-100ms)
        delay := random.Duration(100 * time.Millisecond)
        time.Sleep(delay)
        
        next.ServeHTTP(w, r)
    })
}
```

## Performance

The random package is optimized for high-frequency use:

```
BenchmarkDuration-8      	50000000	    25.3 ns/op
BenchmarkIntBetween-8    	100000000	    12.1 ns/op
BenchmarkChoice-8        	30000000	    38.7 ns/op
BenchmarkAlphaNumeric-8  	5000000	       321 ns/op
BenchmarkShuffle-8       	200000	      7543 ns/op
```

## Thread Safety

All functions in this package are thread-safe and can be called concurrently from multiple goroutines without additional synchronization.

## Security

This package uses Go's `math/rand/v2` which provides cryptographically secure random number generation suitable for security-sensitive applications like:

- Session ID generation
- CSRF token creation
- API key generation
- Cryptographic nonces

## Design Principles

1. **Simple API**: Easy-to-use functions with sensible defaults
2. **Type Safety**: Generic functions that work with any type
3. **Performance**: Efficient implementations for high-frequency use
4. **Predictable**: Clear behavior with well-defined edge cases
5. **Composable**: Functions that work well together

## Contributing

1. Follow the existing code style and patterns
2. Add comprehensive tests for new functionality
3. Include real-world examples in documentation
4. Ensure backward compatibility
5. Update benchmarks for performance-critical changes

## License

This package is part of the core types library and follows the same license terms.
