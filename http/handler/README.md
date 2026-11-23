# handler

`BaseHandler`, an embeddable helper for controllers that wires together
[`request`](../request/) decoding, [`response`](../response/) writing and
error logging.

```go
type UserController struct {
	handler.BaseHandler
	users *UserService
}

func NewUserController(users *UserService) *UserController {
	return &UserController{
		BaseHandler: handler.BaseHandler{}.WithLogger(slog.Default()),
		users:       users,
	}
}

func (c *UserController) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := c.ReadJSON(r, &req); err != nil { // decode + Validate
		c.Next(w, r, err)
		return
	}

	user, err := c.users.Create(r.Context(), req)
	if err != nil {
		c.Next(w, r, err)
		return
	}

	c.Body(w, user, http.StatusCreated) // {"data": {...}}
}
```

| Method | Description |
| --- | --- |
| `ReadJSON(r, v)` | `request.DecodeJSON`: bounded, runs `Bind`/`Validate`. |
| `Next(w, r, err)` | Log `err` (validation errors as warnings) and write `response.ErrorJSON`. |
| `ErrorHandler(fn)` / `ErrorHandlerFunc(fn)` | Adapt `func(w, r) error` handlers; returned errors go to `Next`. |
| `JSON`, `Body`, `WriteRawJSON`, `NoContent`, `ErrorJSON`, `StreamJSON` | Response writers. |
| `QueryValue`, `PathValue`, `FormValue`, `Params` | `request.Value` accessors; `Params` prefers the path value. |
| `ParsePaginationParams(r, def, max)` | `limit`/`offset`/`page` query parameters, clamped and overflow-safe. |
| `ParseSortParams(r, def, allowed)` | `sort`/`order` query parameters restricted to `allowed`. |
| `GetRequestID(r)` / `SetRequestID(w, id)` | Read the ID set by [`requestid`](../requestid/) (or a valid `X-Request-ID`), and echo it. |
| `SetCacheHeaders`, `SetContentType` | Header helpers. |
