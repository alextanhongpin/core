package idempotent

import "context"

type Router struct {
	handlers map[string]Handler
}

func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]Handler),
	}
}

func (r *Router) HandleFunc(pattern string, fn func(ctx context.Context, req []byte) ([]byte, error)) {
	r.Handle(pattern, HandlerFunc(fn))
}

func (r *Router) Handle(pattern string, h Handler) {
	r.handlers[pattern] = h
}

func (r *Router) Handler(pattern string) (Handler, bool) {
	h, ok := r.handlers[pattern]
	return h, ok
}
