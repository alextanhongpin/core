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
	MB                = 1 << 20 // 1 MB
	writeTimeout      = 5 * time.Second
	readTimeout       = 5 * time.Second
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 5 * time.Second
	handlerTimeout    = 5 * time.Second
)

// ListenAndServe starts the HTTP server with some sane defaults.
func ListenAndServe(port string, handler http.Handler) {
	WaitGroup(New(port, handler))
}

// New returns a new server with the default settings.
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
