package httputil

import "net/http"

type Middleware func(http.HandlerFunc) http.HandlerFunc

func Chain(h http.HandlerFunc, mws ...Middleware) http.HandlerFunc {
	for i := len(mws) - 1; i > -1; i-- {
		h = mws[i](h)
	}

	return h
}
