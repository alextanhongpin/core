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

// ListenAndServe runs New(port, handler) with WaitGroup: it blocks until
// SIGINT/SIGTERM or a listen failure, then shuts down gracefully. It returns
// the listen error, if any.
//
// Example:
//
//	if err := server.ListenAndServe(":8080", myHandler); err != nil {
//		log.Fatal(err)
//	}
func ListenAndServe(port string, handler http.Handler) error {
	return WaitGroup(New(port, handler))
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

// WaitGroup runs servers concurrently until SIGINT or SIGTERM is received, or
// until any server fails to listen, and then shuts all of them down,
// allowing in-flight requests up to 5 seconds to finish. It returns the
// listen errors, joined (nil after a clean signal-driven shutdown).
//
// Unless a server already sets BaseContext, request contexts are derived from
// a context that is canceled only after the shutdown grace period, so
// in-flight requests are not canceled the moment the signal arrives.
//
// Example:
//
//	err := server.WaitGroup(
//		server.New(":8080", apiHandler),
//		server.New(":9090", metricsHandler),
//	)
func WaitGroup(servers ...*http.Server) error {
	// SIGINT: Ctrl+C (os.Interrupt is the portable name).
	// SIGTERM: sent by Kubernetes and most process managers on shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	baseCtx, cancelBase := context.WithCancel(context.Background())
	defer cancelBase()

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	// Any listen failure triggers shutdown of the other servers.
	ctx, fail := context.WithCancel(ctx)
	defer fail()

	for _, srv := range servers {
		if srv.BaseContext == nil {
			srv.BaseContext = func(net.Listener) context.Context { return baseCtx }
		}

		wg.Go(func() {
			slog.Default().Info("Server starting", slog.String("addr", srv.Addr))
			err := srv.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Default().Error("Error starting server",
					slog.String("err", err.Error()),
					slog.String("addr", srv.Addr))
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				fail()
				return
			}
			slog.Default().Info("Server stopped", slog.String("addr", srv.Addr))
		})
	}

	<-ctx.Done()

	// Restore default signal behaviour: a second Ctrl+C kills the process.
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	var shutdown sync.WaitGroup
	for _, srv := range servers {
		shutdown.Go(func() {
			if err := srv.Shutdown(shutdownCtx); err != nil {
				slog.Default().Warn("Error shutting down server",
					slog.String("err", err.Error()),
					slog.String("addr", srv.Addr))
			}
		})
	}
	shutdown.Wait()

	// Cancel the requests that outlived the grace period.
	cancelBase()
	wg.Wait()

	return errors.Join(errs...)
}
