package testutil_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alextanhongpin/core/test/testutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDump(t *testing.T) {
	type Person struct {
		Name      string    `json:"name"`
		Age       int64     `json:"age"`
		IsMarried bool      `json:"isMarried"`
		BornAt    time.Time `json:"bornAt"`
	}

	p := Person{
		Name:      "John Appleseed",
		Age:       13,
		IsMarried: true,
		BornAt:    time.Now(),
	}

	// To skip field comparison, return true.
	// This is useful for undeterministic values such ad
	// date time.
	opt := cmpopts.IgnoreMapEntries(func(k string, v any) bool {
		if k == "bornAt" {
			// We just check the expected type can be marshalled
			// to golang's time.
			return testutil.IsJSONTime(t, v)
		}

		// Don't skip by default.
		return false
	})

	// It is recommended to keep the snapshot data in
	// testdata directory.
	if err := testutil.Dump(p, "./testdata/person.json", opt); err != nil {
		t.Error(err)
	}
}

func TestHTTPDump(t *testing.T) {
	type Person struct {
		Name      string    `json:"name"`
		Age       int64     `json:"age"`
		IsMarried bool      `json:"isMarried"`
		BornAt    time.Time `json:"bornAt"`
	}

	p := Person{
		Name:      "John Appleseed",
		Age:       13,
		IsMarried: true,
		BornAt:    time.Now(),
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	r := httptest.NewRequest("GET", "/users/1", nil)

	opt := cmpopts.IgnoreMapEntries(func(k string, v any) bool {
		if k == "bornAt" {
			// We just check the expected type can be marshalled
			// to golang's time.
			return testutil.IsJSONTime(t, v)
		}

		// Don't skip by default.
		return false
	})

	if err := testutil.HTTPDump(r, handler, "./testdata/get_user_response.json", http.StatusOK, opt); err != nil {
		t.Error(err)
	}

	if err := testutil.DotHTTPDump(r, handler, "./testdata/get_user_response.http", http.StatusOK, opt); err != nil {
		t.Error(err)
	}
}
