# middleware

Request logging, panic recovery, and CORS middleware. Each constructor
returns a `func(http.Handler) http.Handler`, so they compose with
[`chain`](../chain/).

```go
logger := slog.Default()

h := chain.Handler(mux,
	middleware.Recovery(logger),
	middleware.Logger(logger),
	middleware.CORS(middleware.CORSConfig{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowCredentials: true,
	}),
)
```

- **`Logger`** logs method, path, remote address, user agent, status, bytes
  written and duration after each request.
- **`Recovery`** logs a panic and replies 500. If the handler already wrote
  the response, the status cannot change. `http.ErrAbortHandler` is
  re-panicked so `net/http` aborts the connection.
- **`CORS`** only adds headers for allowed `Origin`s. Listed origins are
  echoed back with `Vary: Origin` (and credentials, if enabled); `"*"` is
  answered with `Access-Control-Allow-Origin: *` and never with credentials.
  Only real preflights (`OPTIONS` with `Origin` and
  `Access-Control-Request-Method`) are answered with 204; other requests,
  including plain `OPTIONS`, reach the handler.

For rate limiting use
[`sync/ratelimit`](../../sync/ratelimit/)'s `HTTP` middleware.
