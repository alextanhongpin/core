# HTTP Server Package

The HTTP Server package provides production-ready HTTP server utilities with graceful shutdown, zero-downtime deployment support, and sensible defaults for Go web applications.

## Features

- **Graceful Shutdown**: Clean handling of shutdown signals (SIGTERM, SIGINT)
- **Zero-Downtime Deployment**: Support for seamless server upgrades
- **Connection Management**: Proper management of existing connections during shutdown
- **Timeout Configuration**: Sensible defaults for read, write, and idle timeouts
- **Max Request Size**: Protection against oversized payloads
- **Wait Groups**: Built-in waitgroup support for coordinated shutdowns

## Quick Start

```go
package main

import (
    "net/http"
    
    "github.com/alextanhongpin/core/http/server"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("pong"))
    })
    
    // Simple server with graceful shutdown
    server.ListenAndServe(":8080", mux)
    
    // OR use advanced configuration
    srv := server.New(":8080", mux)
    server.WaitGroup(srv)
}
```

## API Reference

### Basic Server

#### `ListenAndServe(addr string, handler http.Handler) error`

Creates and starts an HTTP server with graceful shutdown support.

```go
server.ListenAndServe(":8080", handler)
```

### Advanced Configuration

#### `New(addr string, handler http.Handler, options ...Option) *http.Server`

Creates a new HTTP server with custom configuration options.

```go
srv := server.New(":8080", handler,
    server.WithReadTimeout(5*time.Second),
    server.WithWriteTimeout(10*time.Second),
    server.WithIdleTimeout(120*time.Second),
    server.WithMaxHeaderBytes(1<<20), // 1MB
)

// Start server with graceful shutdown
server.WaitGroup(srv)
```

### Zero-Downtime Deployment

#### `ListenAndServeForever(addr string, handler http.Handler, options ...Option) error`

Starts a server that supports zero-downtime deployments through process inheritance.

```go
server.ListenAndServeForever(":8080", handler)
```

### Graceful Shutdown

#### `WaitGroup(servers ...*http.Server) error`

Waits for shutdown signals and gracefully shuts down provided servers.

```go
// Multiple servers example
apiServer := server.New(":8080", apiHandler)
metricsServer := server.New(":9090", metricsHandler)

// Start servers and wait for shutdown
server.WaitGroup(apiServer, metricsServer)
```

## Configuration Options

### Server Options

```go
// Create a server with custom options
srv := server.New(":8080", handler,
    server.WithReadTimeout(5*time.Second),
    server.WithWriteTimeout(10*time.Second),
    server.WithIdleTimeout(120*time.Second),
    server.WithMaxHeaderBytes(1<<20), // 1MB
    server.WithTLS("cert.pem", "key.pem"),
)
```

### Default Configuration

The server package provides sensible defaults:

- Read Timeout: 5 seconds
- Write Timeout: 10 seconds
- Idle Timeout: 120 seconds
- Max Header Size: 1MB

## Zero-Downtime Deployment

The `forever.go` component enables seamless server upgrades without dropping connections:

```go
server.ListenAndServeForever(":8080", handler)

// To trigger upgrade, send SIGUSR2 to the process
// kill -SIGUSR2 $(lsof -ti:8080)
```

This approach:

1. Keeps the original server running
2. Creates a new process that inherits the socket
3. Starts serving on the new process
4. Gracefully shuts down the old process when all requests are done

## Multiple Servers

```go
package main

import (
    "net/http"
    "github.com/alextanhongpin/core/http/server"
)

func main() {
    // API server
    apiMux := http.NewServeMux()
    apiMux.HandleFunc("/api", apiHandler)
    apiServer := server.New(":8080", apiMux)
    
    // Metrics server
    metricsMux := http.NewServeMux()
    metricsMux.HandleFunc("/metrics", metricsHandler)
    metricsServer := server.New(":9090", metricsMux)
    
    // Start both servers with graceful shutdown
    server.WaitGroup(apiServer, metricsServer)
}
```

## Best Practices

1. **Always Use Graceful Shutdown**: Prevent connection drops and data loss
2. **Configure Timeouts**: Adjust timeouts based on your application needs
3. **Monitor Connections**: Keep track of active connections during shutdown
4. **Load Testing**: Test your server under load to verify graceful shutdown works
5. **TLS Configuration**: Use proper TLS settings in production
