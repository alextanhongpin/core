package ratelimit

import (
	"context"
	"fmt"
	"net/http"
)

type fun[K, V any] func(context.Context, K) (V, error)

type FuncConfig[T any] struct {
	RateLimiter ratelimiter
	KeyFn       func(context.Context, T) (string, error)
}

func Func[K, V any](fn fun[K, V], cfg FuncConfig[K]) fun[K, V] {
	return func(ctx context.Context, req K) (V, error) {
		var zero V
		key, err := cfg.KeyFn(ctx, req)
		if err != nil {
			return zero, err
		}
		if !cfg.RateLimiter.Allow(key) {
			return zero, ErrTooManyRequests
		}

		return fn(ctx, req)
	}
}

func HTTP(next http.Handler, cfg FuncConfig[*http.Request]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, err := cfg.KeyFn(r.Context(), r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res := cfg.RateLimiter.Limit(key)
		if !res.Allow {
			w.Header().Set("Retry-After", res.ResetAfter.String())
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", res.Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))
		w.Header().Set("X-RateLimit-Reset", res.ResetAfter.String())
		next.ServeHTTP(w, r)
	})
}
