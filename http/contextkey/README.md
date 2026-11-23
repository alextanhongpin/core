# contextkey

Typed `context.Context` keys. `Value` returns a `T` instead of `any`, and
every key from `New` is distinct, so two packages that both define a
`"user_id"` key never read each other's values.

```go
import "github.com/alextanhongpin/core/http/contextkey"

var UserIDKey = contextkey.New[string]("user_id")

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := UserIDKey.WithValue(r.Context(), "user-123")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func handler(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDKey.Value(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	fmt.Fprintf(w, "hello, %s", userID)
}
```

## API

| Function | Description |
| --- | --- |
| `New[T](name) *Key[T]` | Create a key. The name is for debugging only. |
| `k.WithValue(ctx, v)` | Return a child context carrying `v`. |
| `k.WithValueNX(ctx, v)` | Like `WithValue`, but keeps an existing value and returns `false`. |
| `k.Value(ctx) (T, bool)` | Read the value. |
| `k.MustValue(ctx) T` | Read the value; panics with an error wrapping `ErrNotFound` if missing. |

Define keys once as package-level variables; a key created inside a
function on each call never matches values stored with another call's key.
