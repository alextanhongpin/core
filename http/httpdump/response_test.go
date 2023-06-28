package httpdump_test

import (
	"bytes"
	"testing"

	"github.com/alextanhongpin/core/http/httpdump"
)

func TestResponseUnmarshalBinary(t *testing.T) {
	b := []byte(`HTTP/1.1 200 OK
Connection: close
Content-Type: text/plain; charset=utf-8

bar`)

	w := new(httpdump.Response)
	if err := w.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}

	got, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	if want := b; !bytes.Equal(want, got) {
		t.Fatalf("want %s, got %s", want, got)
	}
}
