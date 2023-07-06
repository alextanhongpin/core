package httpdump_test

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/http/httpdump"
	"github.com/google/go-cmp/cmp"
)

func TestResponseUnmarshal(t *testing.T) {
	text := []byte(`HTTP/1.1 200 OK
Connection: close
Content-Type: text/plain; charset=utf-8

bar`)
	trailer := []byte(`HTTP/1.1 200 OK
Transfer-Encoding: chunked
Trailer: My-Trailer
Content-Type: application/json
Date: Tue, 04 Jul 2023 15:52:27 GMT

17
{
 "hello": "world"
}
0
My-Trailer: my-val`)

	tests := []struct {
		name string
		data []byte
	}{
		{"text", text},
		{"trailer", trailer},
	}
	for _, ts := range tests {
		name, b := ts.name, ts.data

		t.Run(fmt.Sprintf("text %s", name), func(t *testing.T) {
			w, err := httpdump.ReadResponse(b)
			if err != nil {
				t.Fatal(err)
			}

			got, err := httpdump.DumpResponse(w)
			if err != nil {
				t.Fatal(err)
			}

			want := b
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("want(+), got (-):\n%s", diff)
			}
		})
	}
}
