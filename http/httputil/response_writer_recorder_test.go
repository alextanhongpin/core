package httputil_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/httputil"
	"github.com/stretchr/testify/assert"
)

type handler struct {
}

func (h *handler) ServeHTTP() {

}

func TestResponseWriterRecorder(t *testing.T) {
	mw := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			rw := httputil.NewResponseWriterRecorder(w)
			next.ServeHTTP(rw, r)

			assert := assert.New(t)
			assert.Equal(http.StatusAccepted, rw.StatusCode())
			assert.Equal([]byte("ok"), rw.Body())
		}
		return http.HandlerFunc(fn)
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, "ok")
	})

	wr := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/jobs", nil)

	mw(h).ServeHTTP(wr, r)
}
