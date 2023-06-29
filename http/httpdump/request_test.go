package httpdump_test

import (
	"bytes"
	"testing"

	"github.com/alextanhongpin/core/http/httpdump"
)

func TestRequestUnmarshalText(t *testing.T) {
	b := []byte(`POST /bar HTTP/1.1
Host: 127.0.0.1:61663
User-Agent: Go-http-client/1.1
Content-Length: 17
Accept-Encoding: gzip
Content-Type: application/json

{
 "bar": "baz"
}`)

	r := new(httpdump.Request)
	if err := r.UnmarshalText(b); err != nil {
		t.Fatal(err)
	}

	got, err := r.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	if want := b; !bytes.Equal(want, got) {
		t.Fatalf("want %s, got %s", want, got)
	}
}
