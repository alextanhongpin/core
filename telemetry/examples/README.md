# Telemetry Package - Real World Examples

This directory contains comprehensive real-world examples demonstrating the usage of the telemetry package for structured logging, metrics collection, and observability in Go applications.

## Examples Overview

### 1. Basic Web Server (`main.go`)
A simple HTTP server demonstrating basic telemetry integration with:
- Structured logging with slog
- Prometheus metrics collection
- Request tracking and timing
- Background task monitoring

**How to run:**
```bash
go run main.go
```

Visit http://localhost:8000 for the web interface and http://localhost:8000/metrics for Prometheus metrics.

### 2. Real World Examples Collection (`real_world_examples.go`)
Multiple focused examples showcasing different telemetry scenarios:
- Basic web request processing
- Business logic monitoring
- Error handling and recovery
- Performance monitoring
- Cache operations tracking

**How to run:**
```bash
# Uncomment the main function in real_world_examples.go and run:
go run real_world_examples.go
```

### 3. Advanced E-commerce Service (`ecommerce_advanced.go`)
A comprehensive e-commerce order processing system demonstrating:
- Multi-stage business process telemetry
- Customer tier-based processing
- Payment and inventory management
- Error handling with business context
- HTTP API endpoints with telemetry
- Complex metrics for business intelligence

**Features:**
- Order validation with detailed telemetry
- Customer lookup and tier management
- Inventory checking with retry logic
- Payment processing with fraud detection
- Business rules application
- Order fulfillment tracking
- RESTful API endpoints

**How to run:**
```bash
# Uncomment the main function in ecommerce_advanced.go and run:
go run ecommerce_advanced.go
```

API endpoints available at http://localhost:8081:
- `POST /api/orders` - Create new order
- `GET /api/orders` - List orders
- `GET /api/customers/{id}` - Get customer info
- `GET /api/inventory/{id}` - Check inventory
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

### 4. Microservice Monitoring (`microservice_monitoring.go`)
A distributed system monitoring example featuring:
- Multi-service request tracing
- Circuit breaker pattern implementation
- Service health tracking
- Load testing capabilities
- Real-time service health monitoring

**Features:**
- Distributed request simulation across multiple services
- Authentication, authorization, and business logic services
- Circuit breaker for resilience
- Service health scoring
- Load testing with concurrent requests
- Comprehensive monitoring dashboard

**How to run:**
```bash
# Uncomment the main function in microservice_monitoring.go and run:
go run microservice_monitoring.go
```

Monitoring endpoints available at http://localhost:8082:
- `GET /health` - Overall health status
- `GET /health/services` - Detailed service health
- `GET /api/request?user_id=test` - Simulate distributed request
- `GET /api/load-test` - Run load test
- `GET /metrics` - Prometheus metrics

### 5. Simple Web Server Utilities (`simple_webserver.go`)
Utility functions for creating web servers with built-in telemetry (library functions, not standalone).

## Running All Examples

You can run any of the examples individually by uncommenting the `main()` function in the respective file and running:

```bash
# Basic web server
go run main.go

# Advanced e-commerce service (uncomment main first)
go run ecommerce_advanced.go

# Microservice monitoring (uncomment main first) 
go run microservice_monitoring.go

# Real world examples collection (uncomment main first)
go run real_world_examples.go
```

## Key Telemetry Concepts Demonstrated

### 1. Structured Logging
All examples use structured logging with contextual information:
```go
event.Log(ctx, "order processing started",
    event.String("order_id", order.ID),
    event.String("customer_id", order.CustomerID),
    event.Float64("order_value", order.TotalAmount),
)
```

### 2. Business Metrics
Examples show how to collect meaningful business metrics:
```go
orderCounter := event.NewCounter("orders_processed_total", &event.MetricOptions{
    Description: "Total number of orders processed",
})

orderCounter.Record(ctx, 1,
    event.String("customer_tier", "premium"),
    event.String("status", "success"),
)
```

### 3. Performance Monitoring
Track performance across different operations:
```go
processingTime := event.NewDuration("order_processing_duration", &event.MetricOptions{
    Description: "Time taken to process orders",
})

start := time.Now()
// ... do work ...
processingTime.Record(ctx, time.Since(start),
    event.String("operation", "order_processing"),
)
```

### 4. Error Handling
Comprehensive error tracking with context:
```go
errorCounter := event.NewCounter("business_errors_total", &event.MetricOptions{
    Description: "Business process errors",
})

errorCounter.Record(ctx, 1,
    event.String("error_type", "validation_failed"),
    event.String("service", "order_service"),
)
```

### 5. Health Monitoring
Service health tracking and reporting:
```go
healthScore := event.NewFloatGauge("service_health_score", &event.MetricOptions{
    Description: "Service health score (0-1)",
})

healthScore.Record(ctx, 0.95,
    event.String("service", "payment_service"),
)
```

## Testing the Examples

### Load Testing
Use the microservice monitoring example's load test endpoint:
```bash
curl http://localhost:8082/api/load-test
```

### Creating Orders (E-commerce Example)
```bash
curl -X POST http://localhost:8081/api/orders \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test_order_001",
    "customer_id": "customer_premium_alice",
    "items": [
      {
        "product_id": "laptop_001",
        "name": "Gaming Laptop",
        "quantity": 1,
        "price": 1299.99
      }
    ],
    "total_amount": 1299.99
  }'
```

### Checking Service Health
```bash
curl http://localhost:8082/health/services
```

### Viewing Metrics
All examples expose Prometheus metrics at `/metrics`:
```bash
curl http://localhost:8000/metrics  # Basic web server
curl http://localhost:8081/metrics  # E-commerce service
curl http://localhost:8082/metrics  # Microservice monitoring
```

## Integration with Monitoring Tools

### Prometheus
All examples are configured to work with Prometheus. Add these targets to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'telemetry-basic'
    static_configs:
      - targets: ['localhost:8000']
  
  - job_name: 'telemetry-ecommerce'
    static_configs:
      - targets: ['localhost:8081']
  
  - job_name: 'telemetry-microservices'
    static_configs:
      - targets: ['localhost:8082']
```

### Grafana Dashboards
The metrics can be visualized in Grafana with queries like:
- `rate(microservice_requests_total[5m])` - Request rate
- `histogram_quantile(0.95, rate(microservice_request_duration_bucket[5m]))` - 95th percentile latency
- `microservice_health_score` - Service health scores
- `increase(ecommerce_orders_total[1h])` - Orders per hour

### Log Aggregation
All examples output structured JSON logs that can be ingested by:
- ELK Stack (Elasticsearch, Logstash, Kibana)
- Loki + Grafana
- Fluentd/Fluent Bit
- Any JSON log processor

## Best Practices Demonstrated

1. **Contextual Logging**: Every log entry includes relevant business context
2. **Metric Labeling**: Proper use of labels for dimensional metrics
3. **Error Classification**: Categorizing errors by type and severity
4. **Performance Tracking**: Monitoring latency across different operations
5. **Health Checks**: Implementing comprehensive health monitoring
6. **Circuit Breakers**: Resilience patterns for distributed systems
7. **Business Metrics**: Tracking KPIs alongside technical metrics
8. **Observability**: Making systems observable through telemetry

## Development Notes

- All examples use the same telemetry package but demonstrate different patterns
- Examples are designed to be educational and production-ready
- Error handling demonstrates both technical and business error scenarios
- Metrics are designed to be useful for both operators and business stakeholders
- Examples can be extended for your specific use cases

## Troubleshooting

### Port Conflicts
If you get port binding errors, the ports might be in use. Kill existing processes or modify the port numbers in the code.

### Import Errors
Make sure you're running from the correct directory and that `go mod tidy` has been run in the parent directory.

### Missing Metrics
If metrics don't appear immediately, make a few requests to the applications to generate data.

## Testing Status ✅

All examples have been tested and verified to work correctly:

- ✅ **main.go** - Basic telemetry setup (port 8000)
- ✅ **simple_webserver.go** - Simple web API with metrics (port 8080)  
- ✅ **real_world_examples.go** - Multi-scenario demonstrations
- ✅ **ecommerce_advanced.go** - E-commerce business process tracking (port 8081)
- ✅ **microservice_monitoring.go** - Distributed systems monitoring (port 8082)

Each example demonstrates different aspects of the telemetry package and can be run independently by uncommenting the respective `main()` function.

## Contributing

These examples are meant to be educational. Feel free to extend them for your own use cases or contribute improvements that demonstrate additional telemetry patterns.
