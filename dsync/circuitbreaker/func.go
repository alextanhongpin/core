package circuitbreaker

import "context"

type fun[K, V any] = func(ctx context.Context, req K) (V, error)

type circuitbreaker interface {
	Do(ctx context.Context, key string, fn func() error) error
}

var _ circuitbreaker = (*CircuitBreaker)(nil)

// Func wraps fn so that every call goes through cb. keyFn picks the breaker
// key for a request, for example a constant per dependency or one key per
// tenant. An error from keyFn is returned without calling fn.
//
// The returned function has the same signature as fn, which makes it easy to
// decorate an existing client method:
//
//	getUser := circuitbreaker.Func(api.GetUser, cb,
//		func(ctx context.Context, id string) (string, error) { return "cb:users-api", nil })
//	user, err := getUser(ctx, "42")
func Func[K, V any](fn fun[K, V], cb circuitbreaker, keyFn func(context.Context, K) (string, error)) fun[K, V] {
	return func(ctx context.Context, req K) (res V, err error) {
		key, err := keyFn(ctx, req)
		if err != nil {
			var zero V
			return zero, err
		}
		err = cb.Do(ctx, key, func() error {
			res, err = fn(ctx, req)
			return err
		})
		return
	}
}
