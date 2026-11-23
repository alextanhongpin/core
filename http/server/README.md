# server

`net/http` servers with timeouts and graceful shutdown.

```go
// One server; blocks until SIGINT/SIGTERM or a listen error.
if err := server.ListenAndServe(":8080", mux); err != nil {
	log.Fatal(err)
}

// Several servers sharing one lifecycle.
err := server.WaitGroup(
	server.New(":8080", api),
	server.New(":9090", metrics),
)
```

- `New(addr, h)` returns an `*http.Server` with 5 s read, read-header and
  write timeouts. The write timeout cuts off long-lived responses (SSE,
  large downloads); raise `WriteTimeout` on the returned server for those.
- `WaitGroup(servers...)` starts every server and returns when a signal
  arrives (nil) or any server fails to listen (the joined errors). Either
  way it shuts all servers down, giving in-flight requests 5 s. Request
  contexts stay valid during that grace period and are canceled afterwards
  (unless the server sets its own `BaseContext`).
- `ListenAndServeForever(addr, h)` (Unix only) supports binary upgrades:
  `SIGUSR2` starts a new copy of the executable that binds the same port
  with `SO_REUSEPORT` and sends `SIGQUIT` to the old process (found via the
  `APP_PPID` environment variable), which then drains and exits. `SIGTERM`,
  `SIGQUIT` and `SIGINT` shut down gracefully. Errors are fatal (it calls
  `log.Fatal`). Prefer your orchestrator's rolling restarts when available.
