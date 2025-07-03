// Package server provides production-ready HTTP server utilities with graceful shutdown.
//
// This package simplifies the creation of HTTP servers with sensible defaults
// for production use, including proper timeouts, graceful shutdown handling,
// and signal management.
//
// Key features:
// - Sensible timeout defaults for production use
// - Graceful shutdown on SIGINT/SIGTERM signals
// - Proper error handling and logging
// - Support for running multiple servers concurrently
//
// Example usage:
//
//	// Simple server with default settings
//	server.ListenAndServe(":8080", myHandler)
//
//	// Multiple servers with custom configuration
//	server1 := server.New(":8080", apiHandler)
//	server2 := server.New(":8081", adminHandler)
//	server.WaitGroup(server1, server2)
//
// The servers will automatically handle SIGINT (Ctrl+C) and SIGTERM signals
// for graceful shutdown, allowing in-flight requests to complete before
// terminating the process.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	// MB represents 1 megabyte in bytes
	MB = 1 << 20 // 1 MB
	// writeTimeout is the maximum duration for writing the response
	writeTimeout = 5 * time.Second
	// readTimeout is the maximum duration for reading the request
	readTimeout = 5 * time.Second
	// readHeaderTimeout is the maximum duration for reading request headers
	readHeaderTimeout = 5 * time.Second
	// shutdownTimeout is the maximum time to wait for graceful shutdown
	shutdownTimeout = 5 * time.Second
	// handlerTimeout is the maximum duration for a handler to complete
	handlerTimeout = 5 * time.Second
)

// ListenAndServe starts an HTTP server with default settings and graceful shutdown.
//
// This is a convenience function that creates a new server with sensible defaults
// and waits for it to complete. The server will handle SIGINT and SIGTERM signals
// for graceful shutdown.
//
// Parameters:
//   - port: The port to listen on (e.g., ":8080", ":443")
//   - handler: The HTTP handler to serve requests
//
// This function blocks until the server shuts down or encounters an error.
//
// Example:
//
//	server.ListenAndServe(":8080", myHandler)
func ListenAndServe(port string, handler http.Handler) {
	WaitGroup(New(port, handler))
}

// New creates a new HTTP server with production-ready default settings.
//
// The server is configured with appropriate timeouts for production use:
// - Read timeout: 5 seconds
// - Write timeout: 5 seconds
// - Read header timeout: 5 seconds
//
// These defaults help prevent resource exhaustion and provide good performance
// characteristics for most HTTP services.
//
// Parameters:
//   - port: The port to listen on (e.g., ":8080", "localhost:3000")
//   - handler: The HTTP handler to serve requests
//
// Returns:
//   - A configured *http.Server ready to start
//
// Example:
//
//	server := server.New(":8080", myHandler)
//	log.Fatal(server.ListenAndServe())
//
// Note: The commented-out middleware (TimeoutHandler, MaxBytesHandler) can be
// uncommented if needed, but may interfere with streaming responses or SSE.
func New(port string, handler http.Handler) *http.Server {
	// Instead of setting WriteTimeout, we use http.TimeoutHandler to specify the
	// maximum amount of time for a handler to complete.
	//handler = http.TimeoutHandler(handler, handlerTimeout, "")

	// Also limit the payload size to 1 MB.
	//handler = http.MaxBytesHandler(handler, MaxBytesSize)
	return &http.Server{
		Addr:              port,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		Handler:           handler,
	}
}

// WaitGroup starts multiple HTTP servers concurrently and waits for them to shut down gracefully.
//
// This function handles the lifecycle of multiple HTTP servers:
// 1. Starts all servers in separate goroutines
// 2. Sets up signal handling for SIGINT and SIGTERM
// 3. On signal reception, initiates graceful shutdown of all servers
// 4. Waits for all servers to complete shutdown or timeout
// 5. Logs any errors that occur during startup or shutdown
//
// Parameters:
//   - servers: Variable number of *http.Server instances to run
//
// The function blocks until all servers have shut down or the process is terminated.
//
// Example:
//
//	apiServer := server.New(":8080", apiHandler)
//	adminServer := server.New(":8081", adminHandler)
//	metricsServer := server.New(":9090", metricsHandler)
//
//	server.WaitGroup(apiServer, adminServer, metricsServer)
//
// Signal handling:
// - SIGINT (Ctrl+C): Initiates graceful shutdown
// - SIGTERM: Initiates graceful shutdown (common in container environments)
func WaitGroup(servers ...*http.Server) {
	// SIGINT: When a process is interrupted from keyboard by pressing CTRL+C.
	//         Use os.Interrupt instead for OS-agnostic interrupt.
	//         Reference: https://github.com/edgexfoundry/edgex-go/issues/995
	// SIGTERM: A process is killed. Kubernetes sends this when performing a rolling update.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	wg.Add(len(servers))

	for _, srv := range servers {
		if srv.BaseContext == nil {
			srv.BaseContext = func(_ net.Listener) context.Context {
				// https://www.rudderstack.com/blog/implementing-graceful-shutdown-in-go/
				// Pass the main ctx as the context for every request.
				return ctx
			}
		}

		go func() {
			defer wg.Done()

			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Default().ErrorContext(ctx, "Error starting server",
					slog.String("err", err.Error()),
					slog.String("addr", srv.Addr))
			} else {
				slog.Default().InfoContext(ctx, "Server started",
					slog.String("addr", srv.Addr))
			}
		}()
	}

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behaviour on the interrupt signal and notify user of shutdown.
	stop()

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	for _, srv := range servers {
		if err := srv.Shutdown(ctx); err != nil {
			slog.Default().WarnContext(ctx, "Error shutting down server",
				slog.String("err", err.Error()),
				slog.String("addr", srv.Addr))
		}
	}

	wg.Wait()
}
