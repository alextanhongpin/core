package httpdump_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alextanhongpin/core/http/httpdump"
	"github.com/google/go-cmp/cmp"
)

func TestHTTP(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		body := []byte(`{"data":{"name":"John Appleseed","age":10,"isMarried":true}}`)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}

	r := httptest.NewRequest("POST", "/userinfo", nil)
	rw := httptest.NewRecorder()
	h(rw, r)
	w := rw.Result()

	t.Run("marshal", func(t *testing.T) {
		b, err := httpdump.DumpHTTP(w, r)
		if err != nil {
			t.Fatal(err)
		}

		ww, rr, err := httpdump.ReadHTTP(b)
		if err != nil {
			t.Fatal(err)
		}

		got, err := httpdump.DumpHTTP(ww, rr)
		if err != nil {
			t.Fatal(err)
		}

		want := b
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("want(+), got(-):\n%s", diff)
		}

	})
}
