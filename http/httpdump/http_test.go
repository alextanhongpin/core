package httpdump_test

import (
	"encoding/json"
	"testing"

	"github.com/alextanhongpin/core/http/httpdump"
	"github.com/google/go-cmp/cmp"
)

func TestHTTP(t *testing.T) {
	b := []byte(`POST /bar HTTP/1.1
Host: 127.0.0.1:64011
User-Agent: Go-http-client/1.1
Content-Length: 17
Accept-Encoding: gzip
Content-Type: application/json

{
 "bar": "baz"
}


###


HTTP/1.1 200 OK
Connection: close
Content-Type: text/plain; charset=utf-8

bar`)

	t.Run("text", func(t *testing.T) {
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

	t.Run("json", func(t *testing.T) {
		h1 := new(httpdump.HTTP)
		if err := h1.UnmarshalText(b); err != nil {
			t.Fatal(err)
		}

		b1, err := json.Marshal(h1)
		if err != nil {
			t.Fatal(err)
		}

		var h2 httpdump.HTTP
		if err := json.Unmarshal(b1, &h2); err != nil {
			t.Fatal(err)
		}

		b2, err := h2.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(b1, b2); diff != "" {
			t.Fatalf("want(+), got(-):\n%s", diff)
		}
	})
}
