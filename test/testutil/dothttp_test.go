package testutil_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/test/testutil"
)

func TestDotHTTPDump(t *testing.T) {
	testCases := []struct {
		name       string
		r          *http.Request
		handler    http.HandlerFunc
		statusCode int
	}{
		{
			name: "get_text.http",
			r:    httptest.NewRequest("GET", "/blog.html", nil),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Header().Add("Content-Type", "text/html")
				fmt.Fprint(w, `<!DOCTYPE html><html><head><title>Hello</title></head><body><h1>Hello world</h1></body></html>`)
			},
			statusCode: http.StatusOK,
		},
		{
			name: "get_json.http",
			r:    httptest.NewRequest("GET", "/user.json", nil),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"data": {"name": "John Appleseed", "age": 10, "isMarried": true}}`)
			},
			statusCode: http.StatusOK,
		},
		{
			name: "get_querystring_json.http",
			r: func() *http.Request {
				r := httptest.NewRequest("GET", "/user.json", nil)
				q := r.URL.Query()
				q.Add("limit", "10")
				q.Add("offset", "0")
				q.Add("q", "hello world")
				r.URL.RawQuery = q.Encode()
				return r
			}(),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"data": {"name": "John Appleseed", "age": 10, "isMarried": true}}`)
			},
			statusCode: http.StatusOK,
		},
		{
			name: "post_json.http",
			r: func() *http.Request {
				r := httptest.NewRequest("POST", "/register", strings.NewReader(`{"email": "john.doe@mail.com", "password": "p@$$w0rd"}`))
				r.Header.Set("Content-Type", "application/json;charset=utf-8")
				return r
			}(),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"data": {"accessToken": "@cc3$$T0k3n"}}`)
			},
			statusCode: http.StatusCreated,
		},
		{
			name: "post_form.http",
			r: func() *http.Request {
				form := url.Values{}
				form.Set("username", "john.doe@mail.com")
				form.Set("password", "123456")
				r := httptest.NewRequest("POST", "/register-form", strings.NewReader(form.Encode()))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return r
			}(),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				fmt.Fprint(w, `{"data": {"accessToken": "@cc3$$T0k3n"}}`)
			},
			statusCode: http.StatusCreated,
		},
		{
			name: "delete_no_content.http",
			r:    httptest.NewRequest("DELETE", "/users/1", nil),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
				fmt.Fprint(w, nil)
			},
			statusCode: http.StatusNoContent,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join("./testdata", tc.name)
			if err := testutil.DotHTTPDump(tc.r, tc.handler, out, tc.statusCode); err != nil {
				t.Error(err)
			}
		})
	}
}
