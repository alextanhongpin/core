package httpdump_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/alextanhongpin/core/http/httpdump"
	"github.com/google/go-cmp/cmp"
)

func TestDump(t *testing.T) {
	dump := &httpdump.Dump{
		Line: "POST /bar HTTP/1.1",
		Header: http.Header{
			"Host":            []string{"127.0.0.1:61663"},
			"User-Agent":      []string{"Go-http-client/1.1"},
			"Content-Length":  []string{"17"},
			"Accept-Encoding": []string{"gzip"},
			"Content-Type":    []string{"application/json"},
			"Trailer":         []string{"My-Trailer"},
		},
		Body: bytes.NewReader([]byte(`{"name": "Alice"}`)),
	}

	b, err := json.Marshal(dump)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))

	var d httpdump.Dump
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}

	bb, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(b, bb); err != nil {
		t.Fatalf("want(+), got(-):\n%s", diff)
	}
}
