package middleware

import (
	"cmp"
	"expvar"
	"fmt"
	"net/http"

	"github.com/alextanhongpin/core/http/httputil"
)

var (
	RequestsTotal = expvar.NewMap("requests_total")
	ErrorsTotal   = expvar.NewMap("errors_total")
)

// CounterHandler tracks the success and error count.
// Install the expvar.Handler:
// mux.Handle("GET /debug/vars", expvar.Handler())
func CounterHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wr := httputil.NewResponseWriterRecorder(w)
		h.ServeHTTP(wr, r)

		path := fmt.Sprintf("%s - %d", cmp.Or(r.Pattern, r.URL.Path), wr.StatusCode())
		RequestsTotal.Add("ALL", 1)
		RequestsTotal.Add(path, 1)

		// Treat HTTP status code 2XX as success.
		code := wr.StatusCode()
		if ok := code/100 == 2; !ok {
			ErrorsTotal.Add("ALL", 1)
			ErrorsTotal.Add(path, 1)
		}
	})
}
