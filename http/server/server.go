package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	MaxBytesSize    = 1 << 20 // 1 MB
	readTimeout     = 5 * time.Second
	shutdownTimeout = 5 * time.Second
	handlerTimeout  = 5 * time.Second
)

// ListenAndServe starts the HTTP server with some sane defaults.
func ListenAndServe(port string, handler http.Handler) {

	// SIGINT: When a process is interrupted from keyboard by pressing CTRL+C.
	//         Use os.Interrupt instead for OS-agnostic interrupt.
	//         Reference: https://github.com/edgexfoundry/edgex-go/issues/995
	// SIGTERM: A process is killed. Kubernetes sends this when performing a rolling update.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Instead of setting WriteTimeout, we use http.TimeoutHandler to specify the
	// maximum amount of time for a handler to complete.
	handler = http.TimeoutHandler(handler, handlerTimeout, "")

	// Also limit the payload size to 1 MB.
	handler = http.MaxBytesHandler(handler, MaxBytesSize)
	srv := &http.Server{
		Addr:              port,
		ReadHeaderTimeout: readTimeout,
		ReadTimeout:       readTimeout,
		Handler:           handler,
		BaseContext: func(_ net.Listener) context.Context {
			// https://www.rudderstack.com/blog/implementing-graceful-shutdown-in-go/
			// Pass the main ctx as the context for every request.
			return ctx
		},
	}

	// Initializing the srv in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behaviour on the interrupt signal and notify user of shutdown.
	stop()

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}
}
