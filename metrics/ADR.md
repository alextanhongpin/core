# Architecture Decision Record (ADR): Metrics Collection Package

## Status
Accepted

## Context
We need a comprehensive metrics collection system for Go applications that provides:
- Standard HTTP request metrics (duration, in-flight requests, response size)
- RED (Rate, Error, Duration) metrics for service observability
- Advanced analytics using probabilistic data structures
- Integration with Prometheus for monitoring and alerting
- Thread-safe operations for concurrent environments

## Decision
We will implement a metrics package with the following architecture:

### Core Components

1. **Prometheus Integration Layer**
   - Standard metrics: InFlightGauge, RequestDuration, ResponseSize
   - Custom RED metrics histogram
   - Error counters for detailed error tracking

2. **RED Metrics Tracker**
   - Service-level metrics tracking (Rate, Error, Duration)
   - Thread-safe status management
   - Context-aware cancellation support
   - Panic recovery mechanisms

3. **HTTP Middleware**
   - Request duration tracking
   - In-flight request counting
   - Response size observation
   - Automatic error detection

4. **Advanced Analytics (Tracker)**
   - HyperLogLog for unique visitor counting
   - Count-Min Sketch for frequency estimation
   - T-Digest for quantile estimation
   - TopK for ranking popular endpoints

### Design Principles

1. **Thread Safety**: All metrics operations are thread-safe
2. **Performance**: Minimal overhead for high-throughput applications
3. **Reliability**: Graceful error handling, no panics in metrics code
4. **Observability**: Comprehensive error tracking and debugging support
5. **Flexibility**: Configurable buckets, labels, and collection strategies

## Technical Decisions

### 1. Prometheus as Primary Metrics Backend
**Decision**: Use Prometheus client library for standard metrics
**Rationale**: 
- Industry standard for metrics collection
- Excellent ecosystem support (Grafana, AlertManager)
- Built-in histogram and summary types
- Efficient storage and querying

### 2. Probabilistic Data Structures for Advanced Analytics
**Decision**: Use Redis-backed probabilistic data structures
**Rationale**:
- Constant memory usage regardless of data size
- Approximate results sufficient for analytics
- High performance for real-time processing
- Distributed state management via Redis

### 3. Middleware Pattern for HTTP Instrumentation
**Decision**: Use HTTP middleware for automatic instrumentation
**Rationale**:
- Non-intrusive integration
- Automatic metrics collection
- Easy to add/remove from existing applications
- Composable with other middleware

### 4. Thread-Safe RED Tracker
**Decision**: Implement thread-safe status management with RWMutex
**Rationale**:
- Concurrent access from multiple goroutines
- Read-heavy workload optimization
- Prevent race conditions on status updates

## Implementation Details

### Metrics Schema

```
# In-flight requests
in_flight_requests (gauge)

# Request duration by endpoint
request_duration_seconds (histogram)
- Labels: method, path, status, version

# Response sizes by content type
response_size_bytes (histogram) 
- Labels: content_type

# RED metrics
red_duration_milliseconds (histogram)
- Labels: service, action, status

# Error tracking
errors_total (counter)
- Labels: type, service, action
```

### Edge Cases Handled

1. **Nil Request Handling**: Safe handling of nil HTTP requests
2. **Panic Recovery**: Automatic recovery and metrics recording on panics
3. **Context Cancellation**: Proper status tracking for cancelled operations
4. **Concurrent Access**: Thread-safe metrics updates
5. **Resource Cleanup**: Proper cleanup to prevent memory leaks

### Performance Characteristics

- **RED Tracker Creation**: ~1000ns per operation
- **HTTP Middleware Overhead**: ~5-10μs per request
- **Memory Usage**: Constant for probabilistic structures
- **Concurrent Performance**: Linear scaling with goroutines

## Configuration Options

### Default Configuration
```go
config := MetricsConfig{
    EnableInFlight:     true,
    EnableDuration:     true, 
    EnableResponseSize: true,
    EnableRED:          true,
    EnableErrors:       true,
    Version:           "1.0.0",
    ServiceName:       "unknown",
}
```

### Custom Histogram Buckets
```go
// API latency buckets (optimized for web services)
apiLatencyBuckets := []float64{
    0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

// Response size buckets (bytes to megabytes)
responseSizeBuckets := []float64{
    200, 500, 900, 1500, 5000, 15000, 50000, 150000, 500000,
}
```

## Monitoring and Alerting

### Key Metrics to Monitor

1. **Error Rate**: `rate(red_count{status="err"}[5m]) / rate(red_count[5m])`
2. **Latency**: `histogram_quantile(0.95, rate(request_duration_seconds_bucket[5m]))`
3. **Throughput**: `rate(request_duration_seconds_count[5m])`
4. **In-Flight Requests**: `in_flight_requests`

### Alerting Rules

```yaml
groups:
- name: application_metrics
  rules:
  - alert: HighErrorRate
    expr: rate(red_count{status="err"}[5m]) / rate(red_count[5m]) > 0.1
    for: 5m
    annotations:
      summary: High error rate detected
      
  - alert: HighLatency
    expr: histogram_quantile(0.95, rate(request_duration_seconds_bucket[5m])) > 1
    for: 5m
    annotations:
      summary: High latency detected
```

## Testing Strategy

### Unit Tests
- Basic functionality testing
- Edge case handling
- Concurrent access patterns
- Memory leak detection

### Integration Tests  
- End-to-end HTTP request tracking
- Redis integration for advanced analytics
- Prometheus metrics collection

### Performance Tests
- Benchmark metrics overhead
- Memory allocation profiling
- Concurrent performance validation
- Load testing with realistic workloads

## Migration Strategy

### Phase 1: Basic Metrics
1. Deploy basic HTTP metrics middleware
2. Configure Prometheus collection
3. Set up basic dashboards

### Phase 2: RED Metrics
1. Add RED tracking to critical services
2. Configure service-level alerting
3. Establish SLI/SLO baselines

### Phase 3: Advanced Analytics
1. Deploy Redis infrastructure
2. Enable advanced tracker features
3. Build analytics dashboards

## Consequences

### Positive
- Comprehensive observability across all application layers
- Industry-standard metrics format (Prometheus)
- High performance with minimal overhead
- Advanced analytics capabilities
- Thread-safe concurrent operations

### Negative
- Additional dependency on Redis for advanced features
- Learning curve for probabilistic data structures
- Potential metrics cardinality explosion if not managed properly
- Storage costs for long-term metrics retention

### Risks and Mitigation

1. **High Cardinality Risk**
   - Mitigation: Label validation and cardinality monitoring
   
2. **Performance Impact**
   - Mitigation: Comprehensive benchmarking and performance testing
   
3. **Redis Dependency**
   - Mitigation: Graceful degradation when Redis is unavailable
   
4. **Metrics Storage Costs**
   - Mitigation: Retention policies and downsampling strategies

## Future Considerations

1. **OpenTelemetry Integration**: Consider migration to OpenTelemetry standards
2. **Custom Exporters**: Support for additional metrics backends
3. **Sampling Strategies**: Implement intelligent sampling for high-volume services
4. **Multi-Region Support**: Distributed metrics collection across regions
5. **Machine Learning Integration**: Anomaly detection based on metrics patterns
