# response

JSON response envelopes and writers.

```go
// {"data": {...}}
response.JSON(w, response.Body[User]{Data: user}, http.StatusCreated)

// {"data": [...], "pageInfo": {"hasNextPage": true, ...}}
response.JSON(w, response.Body[[]User]{
	Data:     users,
	PageInfo: &response.PageInfo{HasNextPage: true, EndCursor: cursor},
}, http.StatusOK)

// {"error": {"code": "...", "message": "...", "errors": {...}}}
response.ErrorJSON(w, err)
```

## Functions

| Function | Description |
| --- | --- |
| `JSON(w, v, code)` | Encode `v` first, then write it (`code` 0 means 200). An encoding error gives a generic 500 body instead of a partial response. |
| `ErrorJSON(w, err)` | Map `err` to a status and error body (see below). |
| `RawJSON(w, b, code)` | Write pre-encoded JSON; invalid JSON gives a 500 error. |
| `Text`, `HTML` | Write a string with the matching `Content-Type`. |
| `NoContent(w)` | 204. |
| `SetCacheHeaders(w, maxAge)` | `public, max-age=N`, or no-store when `maxAge <= 0`. |
| `SetSecurityHeaders(w)` | `nosniff`, `X-Frame-Options: DENY`, `X-XSS-Protection`, referrer policy. |
| `Read(resp)` | Read a client `*http.Response` body and restore it (unbounded; only for trusted responses). |
| `NewResponseWriterRecorder(w)` | Wrap a writer to record the status and, with `SetWriteBody(true)`, the full body. |

`ErrorJSON` mapping:

- errors with `Map() map[string][]string` or `Map() map[string]any`
  (validation errors from `github.com/alextanhongpin/errors`) → 400
  `VALIDATION_ERROR` with the field errors;
- `*cause.Error` → `codes.HTTP(code)` with the error's name and message;
- anything else → 500 `INTERNAL_SERVER_ERROR` with a generic message; the
  original error is never exposed.

Errors are matched with `errors.As`, so wrapped and joined errors work.
For CORS use [`middleware.CORS`](../middleware/).
