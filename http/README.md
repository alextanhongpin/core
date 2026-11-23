# HTTP utilities

Small, independent packages for `net/http` services, in one Go module:

```sh
go get github.com/alextanhongpin/core/http
```

| Package | Purpose |
| --- | --- |
| [auth](./auth/) | Basic auth and JWT bearer middleware; JWT signing and verification. |
| [chain](./chain/) | Compose middleware, first listed outermost. |
| [contextkey](./contextkey/) | Typed, collision-free context keys. |
| [handler](./handler/) | `BaseHandler` for controllers: decode, respond, log errors. |
| [health](./health/) | Liveness and readiness handlers with dependency checks. |
| [middleware](./middleware/) | Request logging, panic recovery, CORS. |
| [pagination](./pagination/) | Cursor (keyset) and offset pagination helpers. |
| [request](./request/) | Bounded JSON/form decoding and typed parameter values. |
| [requestid](./requestid/) | Assign and propagate a validated request ID. |
| [response](./response/) | JSON envelopes and error-to-status mapping. |
| [server](./server/) | Servers with timeouts, graceful shutdown, Unix binary upgrades. |
| [templ](./templ/) | `html/template` layouts with optional hot reload. |
| [webhook](./webhook/) | HMAC-SHA256 webhook signing and verification. |

Use the standard library where it already covers a need: Go 1.22+
`http.ServeMux` patterns (`"POST /users/{id}"`, `r.PathValue`) for routing,
`http.MaxBytesHandler` for body limits, and `http.TimeoutHandler` for handler
deadlines. Rate limiting and circuit breaking live in
[`sync/ratelimit`](../sync/ratelimit/) (`ratelimit.HTTP`) and
[`sync/circuitbreaker`](../sync/circuitbreaker/) (`NewTransporter`).

## Example

```go
type UserController struct{ handler.BaseHandler }

func (c *UserController) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest // implements Validate() error
	if err := c.ReadJSON(r, &req); err != nil {
		c.Next(w, r, err) // 400 for validation errors, mapped status for cause errors
		return
	}
	c.Body(w, User{Name: req.Name}, http.StatusCreated)
}

func main() {
	logger := slog.Default()
	users := &UserController{BaseHandler: handler.BaseHandler{}.WithLogger(logger)}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", users.Create)

	h := chain.Handler(mux,
		middleware.Recovery(logger),
		func(next http.Handler) http.Handler {
			return requestid.Handler(next, "X-Request-Id", uuid.NewString)
		},
		middleware.Logger(logger),
		func(next http.Handler) http.Handler {
			return auth.BearerHandler(next, []byte(os.Getenv("JWT_SECRET")))
		},
	)

	if err := server.ListenAndServe(":8080", h); err != nil {
		log.Fatal(err)
	}
}
```

## Verification

```sh
cd http
go vet ./...
go test -race ./...
```
