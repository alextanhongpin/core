package httpdump_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/http/httpdump"
	"github.com/google/go-cmp/cmp"
)

func TestRequestUnmarshal(t *testing.T) {
	p := strings.NewReader(`{"foo": "bar"}`)
	r := httptest.NewRequest("POST", "/", p)

	t.Run("text", func(t *testing.T) {
		b, err := httpdump.DumpRequest(r)
		if err != nil {
			t.Fatal(err)
		}

		rr, err := httpdump.ReadRequest(b)
		if err != nil {
			t.Fatal(err)
		}

		got, err := httpdump.DumpRequest(rr)
		if err != nil {
			t.Fatal(err)
		}

		want := b
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("want(+), got (-):\n%s", diff)
		}
	})
}
