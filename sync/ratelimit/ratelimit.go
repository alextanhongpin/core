package ratelimit

import (
	"sync"
)

type ratelimiter interface {
	Allow() bool
	AllowN(int) bool
}

type RateLimiter struct {
	mu  sync.RWMutex
	rls []ratelimiter
}

func New(rls ...ratelimiter) *RateLimiter {
	return &RateLimiter{
		rls: rls,
	}
}

func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	allow := true
	for _, rl := range r.rls {
		if !rl.Allow() {
			allow = false
		}
	}
	return allow
}

func (r *RateLimiter) AllowN(n int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	allow := true
	for _, rl := range r.rls {
		if !rl.AllowN(n) {
			allow = false
		}
	}
	return allow
}
