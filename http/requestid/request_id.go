// Package requestid provides HTTP middleware that assigns every request an ID.
package requestid

import "net/http"

// MaxLength is the longest client-supplied request ID that Handler keeps.
const MaxLength = 128

// Handler ensures every request carries an ID in the header named key.
//
// A client-supplied ID is kept only if it is at most MaxLength bytes of
// visible ASCII (no spaces or control characters); otherwise it is replaced
// with fn(). This stops clients from injecting oversized or log-breaking
// values. The ID is set on the request header seen by h (the caller's
// request is not modified), on the response header, and in the request
// context under Context.
//
//	h := requestid.Handler(mux, "X-Request-Id", func() string {
//		return uuid.NewString()
//	})
func Handler(h http.Handler, key string, fn func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(key)
		if !Valid(id) {
			id = fn()
		}

		r = r.WithContext(Context.WithValue(r.Context(), id))
		r.Header = r.Header.Clone()
		r.Header.Set(key, id)

		w.Header().Set(key, id)
		h.ServeHTTP(w, r)
	})
}

// Valid reports whether id is a non-empty request ID of at most MaxLength
// visible ASCII characters.
func Valid(id string) bool {
	if id == "" || len(id) > MaxLength {
		return false
	}
	for i := 0; i < len(id); i++ {
		if c := id[i]; c <= ' ' || c > '~' {
			return false
		}
	}
	return true
}
