# requestid

Middleware that gives every request an ID, for log correlation and tracing.

```go
h := requestid.Handler(mux, "X-Request-Id", uuid.NewString)

func handler(w http.ResponseWriter, r *http.Request) {
	id, _ := requestid.Context.Value(r.Context())
	slog.InfoContext(r.Context(), "handling request", "request_id", id)
}
```

`Handler(h, key, fn)`:

- keeps the client's `key` header if `Valid` (non-empty, at most `MaxLength`
  = 128 bytes of visible ASCII); otherwise it generates a new ID with `fn`;
- sets the ID on the request header passed to `h` (the original request is
  not modified), on the response header, and in `requestid.Context`.

A client can still choose any valid ID. If IDs must be unique or
unforgeable, ignore the incoming header and always call `fn` in your own
middleware.
