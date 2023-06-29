package httpdump_test

import (
	"encoding/json"
	"testing"

	"github.com/alextanhongpin/core/http/httpdump"
	"github.com/google/go-cmp/cmp"
)

func TestResponseUnmarshal(t *testing.T) {
	b := []byte(`HTTP/1.1 200 OK
Connection: close
Content-Type: text/plain; charset=utf-8

bar`)

	t.Run("text", func(t *testing.T) {
		r := new(httpdump.Response)
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
		r := new(httpdump.Response)
		if err := r.UnmarshalText(b); err != nil {
			t.Fatal(err)
		}

		j, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}

		var jr httpdump.Response
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
		r := new(httpdump.Response)
		if err := r.UnmarshalText(b); err != nil {
			t.Fatal(err)
		}

		req, err := httpdump.NewResponse(r.Response)
		if err != nil {
			t.Fatal(err)
		}

		j, err := json.Marshal(req)
		if err != nil {
			t.Fatal(err)
		}

		var jr httpdump.Response
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
