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
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := `{"hello": "world"}`
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, body)
	}

	wr := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", strings.NewReader(""))
	handler(wr, r)

	w := wr.Result()

	dump := &testdump.HTTPDump{
		W: w,
		R: r,
	}

	opt := &testdump.HTTPOption{
		Header: []cmp.Option{
			cmpopts.IgnoreMapEntries(func(k string, v any) bool {
				return k == "Host" || k == "Date"
			}),
		},
	}

	hooks := []testdump.Hook[*testdump.HTTPDump]{
		testdump.MaskRequestHeaders(),
	}

	fileName := fmt.Sprintf("testdata/%s.http", t.Name())
	if err := testdump.HTTP(testdump.NewFile(fileName), dump, opt, hooks...); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPTrailer(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Trailer", "my-trailer")
		w.Header().Set("Content-Type", "application/json")
		body := `{"hello": "world"}`
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, body)
		w.Header().Set("my-trailer", "my-val")
	}

	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()

	req, err := http.NewRequest("GET", srv.URL, strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	dump := &testdump.HTTPDump{
		W: resp,
		R: req,
	}

	opt := &testdump.HTTPOption{
		Header: []cmp.Option{
			cmpopts.IgnoreMapEntries(func(k string, v any) bool {
				return k == "Host" || k == "Date"
			}),
		},
	}

	hooks := []testdump.Hook[*testdump.HTTPDump]{
		testdump.MaskRequestHeaders(),
	}

	fileName := fmt.Sprintf("testdata/%s.http", t.Name())
	if err := testdump.HTTP(testdump.NewFile(fileName), dump, opt, hooks...); err != nil {
		t.Fatal(err)
	}
}
