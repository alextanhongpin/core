package snapshottest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alextanhongpin/core/test/snapshottest"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCaptureTest(t *testing.T) {
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
			return snapshottest.IsJsonTime(t, v)
		}

		// Don't skip by default.
		return false
	})
	// It is recommended to keep the snapshot data in
	// testdata directory.
	snapshottest.Capture(t, p, "./testdata/person.json", opt)
}

func TestCaptureHTTPTest(t *testing.T) {
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
			return snapshottest.IsJsonTime(t, v)
		}

		// Don't skip by default.
		return false
	})

	out := "./testdata/get_user_response.json"
	snapshottest.CaptureHTTP(t, r, handler, out, http.StatusOK, opt)
}
