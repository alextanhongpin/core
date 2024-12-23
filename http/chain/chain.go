package chain

import "net/http"

type Middleware func(http.Handler) http.Handler

func Handler(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i > -1; i-- {
		h = mws[i](h)
	}

	return h
}

type MiddlewareFunc func(http.HandlerFunc) http.HandlerFunc

func HandlerFunc(h http.HandlerFunc, mws ...MiddlewareFunc) http.HandlerFunc {
	for i := len(mws) - 1; i > -1; i-- {
		h = mws[i](h)
	}

	return h
}
