// Package server provides production-ready HTTP server utilities with graceful shutdown and zero-downtime upgrades.
package server

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

// ListenAndServeForever allows zero-downtime upgrade using file descriptor inheritance.
//
// This function implements a zero-downtime upgrade mechanism by using Unix signals
// and file descriptor inheritance. When a SIGUSR2 signal is received, the process
// forks a child process that inherits the listener file descriptor, allowing the
// new version to start serving requests while the old version gracefully shuts down.
//
// The upgrade process works as follows:
// 1. The running server listens for SIGUSR2 signals
// 2. On receiving SIGUSR2, it forks a new process with the same binary
// 3. The new process inherits the listener file descriptor
// 4. The old process gracefully shuts down after completing in-flight requests
// 5. The new process continues serving requests without dropping connections
//
// Parameters:
//   - port: The port to listen on (e.g., ":8080")
//   - handler: The HTTP handler to serve requests
//
// Usage example:
//
//	// Start the server
//	go build -o myapp main.go
//	./myapp
//
//	// In another terminal, trigger zero-downtime upgrade:
//	kill -SIGUSR2 $(lsof -ti:8080)
//
//	// Or using process ID:
//	kill -SIGUSR2 <pid>
//
// The function blocks until the server is shut down. It's designed for production
// environments where you need to upgrade the application binary without dropping
// active connections or experiencing downtime.
//
// Requirements:
// - Unix-like operating system (Linux, macOS, etc.)
// - The binary must be accessible at the same path when upgrading
// - Sufficient system resources to run both old and new processes briefly
//
// Note: This function uses low-level Unix system calls and file descriptor
// manipulation, making it unsuitable for Windows environments.
func ListenAndServeForever(port string, handler http.Handler) {
	var l net.Listener

	// Try to obtain parent's listener and kill him.
	if fd, err := listener(port); err != nil {
		l, err = net.Listen("tcp", port)

		if err != nil {
			log.Fatalf("failed to listen to port %s: %v", port, err)
		}
	} else {
		l = fd
		if err := killParent(); err != nil {
			log.Fatalf("failed to kill parent: %v", err)
		}
	}

	// Instead of setting WriteTimeout, we use http.TimeoutHandler to specify the
	// maximum amount of time for a handler to complete.
	// NOTE: Don't use this if you are doing
	// Server-Sent-Events, as this does not implement the
	// Flusher interface.
	//handler = http.TimeoutHandler(handler, handlerTimeout, "")

	// Also limit the payload size to 1 MB.
	//handler = http.MaxBytesHandler(handler, MB)

	// Start the web server.
	s := &http.Server{
		ReadHeaderTimeout: readTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		Handler:           handler,
	}

	go func() {
		if err := s.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to close serve: %v", err)
		}
	}()

	// Start loop which is responsible for upgrade watching.
	upgradeLoop(s)
}

func upgradeLoop(s *http.Server) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGQUIT, syscall.SIGUSR2, syscall.SIGTERM, os.Interrupt)
	for t := range sig {
		switch t {
		case syscall.SIGUSR2:
			// Fork a child and start binary upgrading.
			if err := spawnChild(); err != nil {
				log.Println(
					"Cannot perform binary upgrade, when starting process: ",
					err.Error(),
				)
				continue
			}
		case syscall.SIGQUIT, syscall.SIGTERM, os.Interrupt:
			ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()

			if err := s.Shutdown(ctx); err != nil {
				log.Fatal(err)
			}

			os.Exit(0)
			return
		}
	}
}

func listener(port string) (net.Listener, error) {
	lc := net.ListenConfig{
		Control: control,
	}
	if l, err := lc.Listen(context.Background(), "tcp", port); err != nil {
		return nil, err
	} else {
		return l, nil
	}
}

// When parent process exists, send it signals, that it should perform graceful
// shutdown and stop serving new requests.
func killParent() error {
	pid, ok := os.LookupEnv("APP_PPID")
	if !ok {
		return nil
	}

	ppid, err := strconv.Atoi(pid)
	if err != nil {
		return err
	}

	if p, err := os.FindProcess(ppid); err != nil {
		return err
	} else {
		return p.Signal(syscall.SIGQUIT)
	}
}

func spawnChild() error {
	argv0, err := exec.LookPath(os.Args[0])
	if err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	files := make([]*os.File, 0)
	files = append(files, os.Stdin, os.Stdout, os.Stderr)

	ppid := os.Getpid()
	os.Setenv("APP_PPID", strconv.Itoa(ppid))

	_, err = os.StartProcess(argv0, os.Args, &os.ProcAttr{
		Dir:   wd,
		Env:   os.Environ(),
		Files: files,
		Sys:   &syscall.SysProcAttr{},
	})

	return err
}

func control(network, address string, c syscall.RawConn) error {
	var err error
	cerr := c.Control(func(fd uintptr) {
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			return
		}

		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		if err != nil {
			return
		}
	})

	return errors.Join(cerr, err)
}
