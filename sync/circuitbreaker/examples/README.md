# Circuit Breaker Examples

This directory contains comprehensive examples demonstrating the circuit breaker package functionality.

## Examples

### 1. Simple Example (`simple/main.go`)
A basic example showing circuit breaker functionality with an unreliable service.

```bash
go run simple/main.go
```

**Features demonstrated:**
- Basic circuit breaker configuration
- State transitions (closed -> open -> half-open -> closed)
- Error handling and recovery
- State change callbacks

### 2. Advanced Example (`main.go`)
A comprehensive example with metrics, callbacks, and multiple failure scenarios.

```bash
go run main.go
```

**Features demonstrated:**
- Custom configuration options
- Metrics collection and monitoring
- Multiple failure scenarios (high failure rate, timeouts, recovery)
- HTTP client integration
- Concurrent request handling
- State transition monitoring

### 3. HTTP Client Example (`http/main.go`)
Real-world HTTP client integration with circuit breaker.

```bash
go run http/main.go
```

**Features demonstrated:**
- HTTP client integration
- API server simulation
- Concurrent request handling
- Error recovery patterns
- Production-ready patterns

## Running All Examples

To run all examples, use:

```bash
# Simple example
go run simple/main.go

# Advanced example  
go run main.go

# HTTP client example
go run http/main.go
```

## Key Concepts Demonstrated

1. **State Machine**: Shows how the circuit breaker transitions between closed, open, and half-open states
2. **Failure Threshold**: Demonstrates how failure counts trigger state changes
3. **Recovery**: Shows how the circuit breaker recovers when service improves
4. **Monitoring**: Examples of state change callbacks and metrics collection
5. **Real-world Integration**: HTTP client patterns and error handling
6. **Concurrency**: Safe concurrent usage of the circuit breaker

## Output Examples

Each example produces colored output showing:
- ✅ Successful operations
- ❌ Failed operations  
- ⚡ Circuit breaker blocked requests
- 🔄 State transitions
- 📊 Metrics and statistics

This makes it easy to understand how the circuit breaker behaves in different scenarios.
