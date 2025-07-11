# HTTP Server Package

The HTTP Server package provides production-ready HTTP server utilities with graceful shutdown, zero-downtime deployment support, and sensible defaults for Go web applications.

## Features

- **Graceful Shutdown**: Clean handling of shutdown signals (SIGTERM, SIGINT)
- **Zero-Downtime Deployment**: Support for seamless server upgrades
- **Connection Management**: Proper management of existing connections during shutdown
- **Timeout Configuration**: Sensible defaults for read, write, and idle timeouts
- **Max Request Size**: Protection against oversized payloads
- **Wait Groups**: Built-in waitgroup support for coordinated shutdowns
- **Cross-Package Integration**: Works with `health`, `chain`, and middleware packages

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

### Advanced Configuration

#### `New(addr string, handler http.Handler, options ...Option) *http.Server`
Creates a new HTTP server with custom configuration options.

#### `WaitGroup(srv *http.Server)`
Waits for server shutdown and coordinates cleanup.

## Best Practices

- Always use graceful shutdown for production servers.
- Configure timeouts and max request size for security and reliability.
- Integrate health endpoints for orchestration and monitoring.

## Related Packages

- [`health`](../health/README.md): Health check endpoints
- [`chain`](../chain/README.md): Middleware chaining
- [`middleware`](../middleware/README.md): Middleware components

## License

MIT
