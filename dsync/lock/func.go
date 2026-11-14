package lock

import "context"

// Doer is anything with Locker's Do method, such as *Locker.
type Doer interface {
	Do(ctx context.Context, key string, fn func(ctx context.Context) error) error
}

// Func wraps fn so that every call runs inside l.Do, under the key returned by
// keyFn for that request. The wrapped function returns fn's result even when
// the error is non-nil (for example, fn succeeded but the release failed). If
// keyFn fails, fn is not called.
func Func[K, V any](fn func(ctx context.Context, req K) (V, error), l Doer, keyFn func(context.Context, K) (string, error)) func(ctx context.Context, req K) (V, error) {
	return func(ctx context.Context, req K) (V, error) {
		var zero V
		key, err := keyFn(ctx, req)
		if err != nil {
			return zero, err
		}
		var res V
		err = l.Do(ctx, key, func(ctx context.Context) error {
			var fnErr error
			res, fnErr = fn(ctx, req)
			return fnErr
		})
		return res, err
	}
}
