# chain

Compose middleware so the first one listed is the outermost.

```go
h := chain.Handler(mux,
	middleware.Recovery(logger), // runs first
	middleware.Logger(logger),
	authMiddleware,              // runs last, right before mux
)
// same as: Recovery(Logger(authMiddleware(mux)))
```

- `Handler(h, mws...)` takes `func(http.Handler) http.Handler` middleware.
- `HandlerFunc(h, mws...)` does the same for
  `func(http.HandlerFunc) http.HandlerFunc`.

With no middleware the handler is returned unchanged.
