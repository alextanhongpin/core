package middleware

import "net/http"

func RequestIDHandler(h http.Handler, key string, fn func() (string, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(key)
		if id == "" {
			newID, err := fn()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			id = newID
			r.Header.Set(key, id)
		}

		w.Header().Set(key, id)
		h.ServeHTTP(w, r)
	})
}
