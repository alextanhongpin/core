package requestid

import "net/http"

func Handler(h http.Handler, key string, fn func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		id := r.Header.Get(key)
		if id == "" {
			id = fn()
			r.Header.Set(key, id)

		}

		w.Header().Set(key, id)
		h.ServeHTTP(w, r.WithContext(Context.WithValue(r.Context(), id)))
	})
}
