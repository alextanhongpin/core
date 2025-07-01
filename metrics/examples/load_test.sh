#!/bin/bash

# Load testing script for the metrics package examples

echo "🚀 Starting E-commerce API load test..."

# Base URL
BASE_URL="http://localhost:8080"

# Check if server is running
if ! curl -s "${BASE_URL}/health" > /dev/null; then
    echo "❌ Server is not running. Please start the server first:"
    echo "   go run examples/ecommerce/main.go"
    exit 1
fi

echo "✅ Server is running"

# Function to make concurrent requests
make_requests() {
    local endpoint=$1
    local method=${2:-GET}
    local data=${3:-""}
    local count=${4:-10}
    
    echo "📡 Making $count $method requests to $endpoint..."
    
    for i in $(seq 1 $count); do
        if [ "$method" = "POST" ] && [ -n "$data" ]; then
            curl -s -X POST "${BASE_URL}${endpoint}" \
                -H "Content-Type: application/json" \
                -H "X-User-ID: user-$((i % 5 + 1))" \
                -d "$data" > /dev/null &
        else
            curl -s -H "X-User-ID: user-$((i % 5 + 1))" "${BASE_URL}${endpoint}" > /dev/null &
        fi
        
        # Add some randomness
        if [ $((i % 10)) -eq 0 ]; then
            sleep 0.1
        fi
    done
    
    wait
    echo "✅ Completed $count requests to $endpoint"
}

# Create users
echo "👥 Creating test users..."
for i in {1..5}; do
    curl -s -X POST "${BASE_URL}/api/users/create" \
        -H "Content-Type: application/json" \
        -d "{\"name\":\"User$i\",\"email\":\"user$i@example.com\"}" > /dev/null &
done
wait

# Generate load on different endpoints
echo "🔥 Generating load..."

# List users (light load)
make_requests "/api/users" "GET" "" 50 &

# List orders (medium load)  
make_requests "/api/orders" "GET" "" 30 &

# Create orders (heavy load with potential failures)
make_requests "/api/orders/create" "POST" '{"user_id":1,"amount":99.99}' 20 &

# Process orders (slow endpoint)
make_requests "/api/orders/process?id=1" "POST" "" 10 &

# Load test endpoint (variable latency)
make_requests "/api/load-test" "GET" "" 100 &

echo "⏳ Waiting for all requests to complete..."
wait

echo "📊 Load test completed! Check metrics at:"
echo "   - Prometheus metrics: ${BASE_URL}/metrics"
echo "   - Analytics: ${BASE_URL}/admin/analytics"

# Display some metrics
echo ""
echo "📈 Sample metrics:"
curl -s "${BASE_URL}/metrics" | grep -E "(in_flight_requests|request_duration_seconds_count|red_count)" | head -10

echo ""
echo "🎯 To view in Prometheus/Grafana:"
echo "   1. Start Prometheus: prometheus --config.file=prometheus.yml"
echo "   2. Add target: localhost:8080"
echo "   3. Query: rate(request_duration_seconds_count[5m])"
echo "   4. Query: histogram_quantile(0.95, rate(request_duration_seconds_bucket[5m]))"
