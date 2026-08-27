package ratelimit

import (
	"context"
	"fmt"
	"net/http"
)

type handler[K, V any] func(context.Context, K) (V, error)

type HandlerConfig[T any] struct {
	RateLimiter ratelimiter
	NewKey      func(context.Context, T) (string, error)
}

func Handler[K, V any](fn handler[K, V], cfg HandlerConfig[K]) handler[K, V] {
	return func(ctx context.Context, req K) (V, error) {
		var zero V
		key, err := cfg.NewKey(ctx, req)
		if err != nil {
			return zero, err
		}
		if !cfg.RateLimiter.Allow(key) {
			return zero, ErrTooManyRequests
		}

		return fn(ctx, req)
	}
}

func HTTP(next http.Handler, cfg HandlerConfig[*http.Request]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, err := cfg.NewKey(r.Context(), r)
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
