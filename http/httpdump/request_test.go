package httpdump_test

import (
	"encoding/json"
	"testing"

	"github.com/alextanhongpin/core/http/httpdump"
	"github.com/google/go-cmp/cmp"
)

func TestRequestUnmarshal(t *testing.T) {
	b := []byte(`POST /bar HTTP/1.1
Host: 127.0.0.1:61663
User-Agent: Go-http-client/1.1
Content-Length: 17
Accept-Encoding: gzip
Content-Type: application/json

{
 "bar": "baz"
}`)

	t.Run("text", func(t *testing.T) {
		r := new(httpdump.Request)
		if err := r.UnmarshalText(b); err != nil {
			t.Fatal(err)
		}

		got, err := r.MarshalText()
		if err != nil {
			t.Fatal(err)
		}

		want := b
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("want(+), got (-):\n%s", diff)
		}
	})

	t.Run("json", func(t *testing.T) {
		r := new(httpdump.Request)
		if err := r.UnmarshalText(b); err != nil {
			t.Fatal(err)
		}

		j, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}

		var jr httpdump.Request
		if err := json.Unmarshal(j, &jr); err != nil {
			t.Fatal(err)
		}

		got, err := jr.MarshalText()
		if err != nil {
			t.Fatal(err)
		}

		want := b
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("want(+), got (-):\n%s", diff)
		}
	})

	t.Run("parse", func(t *testing.T) {
		r := new(httpdump.Request)
		if err := r.UnmarshalText(b); err != nil {
			t.Fatal(err)
		}

		dr := httpdump.NewRequest(r.Request)
		if err := dr.Parse(); err != nil {
			t.Fatal(err)
		}

		j, err := json.Marshal(dr)
		if err != nil {
			t.Fatal(err)
		}

		var jr httpdump.Request
		if err := json.Unmarshal(j, &jr); err != nil {
			t.Fatal(err)
		}

		got, err := jr.MarshalText()
		if err != nil {
			t.Fatal(err)
		}

		want := b
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("want(+), got (-):\n%s", diff)
		}
	})
}
