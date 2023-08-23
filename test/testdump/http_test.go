package testdump_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestHTTP(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Trailer", "my-trailer")
		w.Header().Set("Content-Type", "application/json")
		body := `{"hello": "world"}`
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, body)
		w.Header().Set("my-trailer", "my-val")
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(h))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r, err := http.NewRequest("GET", ts.URL, strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}

	fileName := fmt.Sprintf("testdata/%s.http", t.Name())
	dump := &testdump.HTTPDump{
		W: resp,
		R: r,
	}

	if err := testdump.HTTP(testdump.NewFile(fileName), dump, &testdump.HTTPOption{
		Header: []cmp.Option{
			cmpopts.IgnoreMapEntries(func(k string, v any) bool {
				return k == "Host" || k == "Date"
			}),
		},
		Hooks: []testdump.Hook[*testdump.HTTPDump]{
			testdump.MaskRequestHeaders(),
		},
	}); err != nil {
		t.Fatal(err)
	}
}
