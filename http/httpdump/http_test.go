package httpdump_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/httpdump"
	"github.com/google/go-cmp/cmp"
)

func TestHTTP(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"data": {"name": "John Appleseed", "age": 10, "isMarried": true}}`)
	}

	r := httptest.NewRequest("POST", "/userinfo", nil)
	w := httptest.NewRecorder()
	h(w, r)
	resp := w.Result()

	t.Run("marshal", func(t *testing.T) {
		d, err := httpdump.NewHTTP(resp, r)
		if err != nil {
			t.Fatal(err)
		}
		b, err := d.MarshalText()
		if err != nil {
			t.Fatal(err)
		}

		h := new(httpdump.HTTP)
		if err := h.UnmarshalText(b); err != nil {
			t.Fatal(err)
		}

		got, err := h.MarshalText()
		if err != nil {
			t.Fatal(err)
		}

		want := b
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("want(+), got(-):\n%s", diff)
		}

	})
}
