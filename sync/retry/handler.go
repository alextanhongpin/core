package retry

import (
	"context"
	"errors"
)

func Do(ctx context.Context, fn func(context.Context) error, n int) (err error) {
	return New().Do(ctx, fn, n)
}

func DoValue[T any](ctx context.Context, fn func(context.Context) (T, error), n int) (v T, err error) {
	r := New()
	for _, retryErr := range r.Try(ctx, n) {
		if retryErr != nil {
			return v, errors.Join(retryErr, err)
		}

		v, err = fn(ctx)
		if err == nil {
			break
		}
	}

	return
}
