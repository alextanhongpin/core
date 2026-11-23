# request

Bounded request body decoding and typed access to query, path, form and
header values.

```go
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (r *CreateUserRequest) Validate() error { /* ... */ }

func createUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := request.DecodeJSON(r, &req, request.WithRequired()); err != nil {
		response.ErrorJSON(w, err)
		return
	}
	// or: req, err := request.Parse[CreateUserRequest](r)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	id, err := request.PathValue(r, "id").Int64Range(1, math.MaxInt64)
	limit := request.QueryValue(r, "limit").IntRangeOr(1, 100, 20)
	tags := request.QueryValue(r, "tags").CSV()
	// ...
}
```

## Bodies

- `DecodeJSON(r, v, opts...)` reads at most `MaxBodySize` (10 MiB;
  `WithMaxBodySize` overrides) and fails with `ErrBodyTooLarge` beyond it.
  An empty body fails with `ErrEmptyBody` under `WithRequired`, otherwise
  `v` keeps its zero value. `v.Bind(r)` and `v.Validate()` run after
  decoding when implemented, also for an empty body. Decoding errors are
  `*BodyError` (with the raw body) wrapping `ErrInvalidJSON`, `ErrBodyTooLarge`
  or the JSON error. The body is restored for later readers.
- `Parse[T](r, opts...)` is `DecodeJSON` into a new `T`.
- `DecodeForm` / `DecodeQuery` bind `form` (or `json`) tagged fields of a
  struct pointer from form or query values, then call `Validate`. Numbers are
  parsed at the field's size, so out-of-range values are errors.
- `Read(r)` / `Clone(r)` buffer the body (up to `MaxBodySize`) so it can be
  read again.

## Values

`QueryValue`, `QueryValues`, `PathValue`, `FormValue` and `HeaderValue`
return `Value`, a string with helpers:

- conversion: `Int`, `Int32`, `Int64`, `Float64`, `Bool` return the zero
  value for invalid input;
- defaults: `IntOr`, `Int64Or`, `Float64Or`, `BoolOr`, `StringOr`, ... return
  the default only for empty or invalid input (an explicit `0` is kept);
- validation: `IntRange`, `Float64Range`, ... return an error for invalid,
  out-of-range, NaN or infinite input; `Required`, `Length`, `Email`;
- `Trim`, `Lower`, `Upper`, `CSV`, `Split`, `Time`, `Date`, `RFC3339`, `URL`,
  `FromBase64`, `ToBase64`, `Match`, `InSlice`.
