package testutil_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alextanhongpin/core/test/testutil"
)

func TestHTTPDump(t *testing.T) {
	testCases := []struct {
		name    string
		r       *http.Request
		handler http.HandlerFunc
		opts    []testutil.Option
	}{
		{
			name: "get html",
			r:    httptest.NewRequest("GET", "/blog.html", nil),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Header().Add("Content-Type", "text/html")
				fmt.Fprint(w, `<!DOCTYPE html><html><head><title>Hello</title></head><body><h1>Hello world</h1></body></html>`)
			},
		},
		{
			name: "get json",
			r:    httptest.NewRequest("GET", "/user.json", nil),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"data": {"name": "John Appleseed", "age": 10, "isMarried": true}}`)
			},
		},
		{
			name: "get query string",
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
		},
		{
			name: "post json",
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
		},
		{
			name: "post form",
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
		},
		{
			name: "delete no content",
			r:    httptest.NewRequest("DELETE", "/users/1", nil),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
				fmt.Fprint(w, nil)
			},
		},
		{
			name: "skip fields",
			r: func() *http.Request {
				r := httptest.NewRequest("POST", "/songs", strings.NewReader(`{"title": "onespecies"}`))
				r.Header.Set("Content-Type", "application/json;charset=utf-8")
				return r
			}(),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				w.Header().Set("Content-Type", "application/json")
				id := rand.Int63()
				fmt.Fprintf(w, `{"data": {"id": %d}}`, id)
			},
			opts: []testutil.Option{
				testutil.IgnoreFields("id"),
			},
		},
		{
			name: "inspect body",
			r: func() *http.Request {
				r := httptest.NewRequest("POST", "/songs", strings.NewReader(`{"title": "onespecies"}`))
				r.Header.Set("Content-Type", "application/json;charset=utf-8")
				return r
			}(),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"data": {"createdAt": %q}}`, time.Now().Format(time.RFC3339))
			},
			opts: []testutil.Option{
				testutil.IgnoreFields("createdAt"),
				testutil.InspectBody(func(body []byte) {
					type response struct {
						Data struct {
							CreatedAt time.Time `json:"createdAt"`
						} `json:"data"`
					}

					var res response
					if err := json.Unmarshal(body, &res); err != nil {
						t.Fatal(err)
					}

					if res.Data.CreatedAt.IsZero() {
						t.Fatalf("want createdAt to be non-zero, got %s", res.Data.CreatedAt)
					}
				}),
			},
		},
		{
			name: "inspect header",
			r: func() *http.Request {
				r := httptest.NewRequest("DELETE", "/songs/1", nil)
				r.Header.Set("Content-Type", "application/json;charset=utf-8")
				return r
			}(),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, nil)
			},
			opts: []testutil.Option{
				testutil.InspectHeaders(func(headers http.Header) {
					contentType, params, err := mime.ParseMediaType(headers.Get("Content-Type"))
					if err != nil {
						t.Fatal(err)
					}
					if params != nil {
						t.Fatalf("want params to be nil, got %v", params)
					}
					if contentType != "application/json" {
						t.Fatalf("want content-type to be application/json, got %s", contentType)
					}
				}),
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testutil.DumpHTTP(t, tc.r, tc.handler, tc.opts...)
		})
	}
}

func TestHTTP(t *testing.T) {
	fooHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "foo")
	}
	barHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "bar")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/foo", fooHandler)
	mw := testutil.DumpHTTPHandler(t, testutil.IgnoreHeaders("Host"))
	mux.Handle("/bar", mw(http.HandlerFunc(barHandler)))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx := context.Background()
	client := &YourClient{url: ts.URL}

	foo, err := client.GetFoo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if foo != "foo" {
		t.Fatalf("want %s, got %s", "foo", foo)
	}

	bar, err := client.GetBar(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if bar != "bar" {
		t.Fatalf("want %s, got %s", "bar", bar)
	}
}

type YourClient struct {
	url string
}

func (c *YourClient) GetFoo(ctx context.Context) (string, error) {
	endpoint, err := url.JoinPath(c.url, "/foo")
	if err != nil {
		return "", err
	}

	fmt.Println("calling", endpoint)
	resp, err := http.Get(endpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (c *YourClient) GetBar(ctx context.Context) (string, error) {
	endpoint, err := url.JoinPath(c.url, "/bar")
	if err != nil {
		return "", err
	}

	fmt.Println("calling", endpoint)
	resp, err := http.Get(endpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
