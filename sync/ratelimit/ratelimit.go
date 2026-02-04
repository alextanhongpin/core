package ratelimit

type ratelimiter interface {
	Allow() bool
	AllowN(int) bool
}

type RateLimiter struct {
	rls []ratelimiter
}

func New(rls ...ratelimiter) *RateLimiter {
	return &RateLimiter{
		rls: rls,
	}
}

func (r *RateLimiter) Allow() bool {
	allow := true
	for _, rl := range r.rls {
		if !rl.Allow() {
			allow = false
		}
	}
	return allow
}

func (r *RateLimiter) AllowN(n int) bool {
	allow := true
	for _, rl := range r.rls {
		if !rl.AllowN(n) {
			allow = false
		}
	}
	return allow
}
