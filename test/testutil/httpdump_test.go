package testutil_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	htmlHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		// NOTE: WriteHeader must be called after Header.
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `<!DOCTYPE html><html><head><title>Hello</title></head><body><h1>Hello world</h1></body></html>`)
	}

	jsonHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"data": {"name": "John Appleseed", "age": 10, "isMarried": true}}`)
	}
	loginHandler := func(w http.ResponseWriter, r *http.Request) {
		type loginRequest struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("WWW-Authenticate", "Basic realm=<realm>, charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"data": {"accessToken": "@cc3$$T0k3n"}}`)
	}

	registerHandler := func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		email := r.Form.Get("email")
		password := r.Form.Get("password")
		if email != "john.doe@mail.com" && password != "123456" {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"data": {"accessToken": "@cc3$$T0k3n"}}`)
	}

	noContentHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		fmt.Fprint(w, nil)
	}

	testCases := []struct {
		name    string
		r       *http.Request
		handler http.HandlerFunc
		opts    []testutil.HTTPOption
	}{
		{
			name:    "get html",
			r:       httptest.NewRequest("GET", "/blog.html", nil),
			handler: htmlHandler,
		},
		{
			name:    "get json",
			r:       httptest.NewRequest("GET", "/user.json", nil),
			handler: jsonHandler,
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
			handler: jsonHandler,
		},
		{
			name: "post bearer",
			r: func() *http.Request {
				r := httptest.NewRequest("POST", "/register", strings.NewReader(`{"email": "john.doe@mail.com", "password": "p@$$w0rd"}`))
				r.Header.Set("Content-Type", "application/json;charset=utf-8")
				r.Header.Set("Authorization", "Bearer xyz")
				return r
			}(),
			opts: []testutil.HTTPOption{
				testutil.MaskRequestHeaders("Authorization"),
				testutil.MaskResponseHeaders("WWW-Authenticate"),
			},
			handler: loginHandler,
		},
		{
			name: "post json",
			r: func() *http.Request {
				r := httptest.NewRequest("POST", "/register", strings.NewReader(`{"email": "john.doe@mail.com", "password": "p@$$w0rd"}`))
				r.Header.Set("Content-Type", "application/json;charset=utf-8")
				return r
			}(),
			handler: loginHandler,
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
			handler: registerHandler,
		},
		{
			name:    "delete no content",
			r:       httptest.NewRequest("DELETE", "/users/1", nil),
			handler: noContentHandler,
		},
		{
			name: "skip fields",
			r: func() *http.Request {
				r := httptest.NewRequest("POST", "/songs", strings.NewReader(`{"title": "onespecies"}`))
				r.Header.Set("Content-Type", "application/json;charset=utf-8")
				return r
			}(),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				id := rand.Int63()
				fmt.Fprintf(w, `{"data": {"id": %d}}`, id)
			},
			opts: []testutil.HTTPOption{
				testutil.IgnoreBodyFields("id"),
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
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				fmt.Fprintf(w, `{"data": {"createdAt": %q}}`, time.Now().Format(time.RFC3339))
			},
			opts: []testutil.HTTPOption{
				testutil.IgnoreBodyFields("createdAt"),
				testutil.InspectResponseBody(func(body []byte) error {
					type response struct {
						Data struct {
							CreatedAt time.Time `json:"createdAt"`
						} `json:"data"`
					}

					var res response
					if err := json.Unmarshal(body, &res); err != nil {
						return err
					}

					if res.Data.CreatedAt.IsZero() {
						return fmt.Errorf("want createdAt to be non-zero, got %s", res.Data.CreatedAt)
					}

					return nil
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
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNoContent)
				fmt.Fprint(w, nil)
			},
			opts: []testutil.HTTPOption{
				testutil.InspectRequestHeaders(func(headers http.Header) {
					contentType, params, err := mime.ParseMediaType(headers.Get("Content-Type"))
					if err != nil {
						t.Fatal(err)
					}
					if want, got := "utf-8", params["charset"]; want != got {
						t.Fatalf("want %s, got %s", want, got)
					}
					if want, got := "application/json", contentType; want != got {
						t.Fatalf("want %s, got %s", want, got)
					}
				}),
			},
		},
		{
			name: "inspect multi header",
			r: func() *http.Request {
				r := httptest.NewRequest("GET", "/search?q=John&limit=10", nil)
				r.Header.Add("Cache-Control", "max-age=604800")
				r.Header.Add("Cache-Control", "stale-while-revalidate=86400")
				r.Header.Set("Content-Type", "application/json;charset=utf-8")
				return r
			}(),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNoContent)
				fmt.Fprint(w, nil)
			},
			opts: []testutil.HTTPOption{
				testutil.InspectRequestHeaders(func(headers http.Header) {
					cacheControl := headers["Cache-Control"]

					maxAge := "max-age=604800"
					staleWhileRevalidate := "stale-while-revalidate=86400"
					if want, got := maxAge, cacheControl[0]; want != got {
						t.Fatalf("want %s, got %s", want, got)
					}

					if want, got := staleWhileRevalidate, cacheControl[1]; want != got {
						t.Fatalf("want %s, got %s", want, got)
					}
				}),
			},
		},
		{
			name: "masked request",
			r: func() *http.Request {
				r := httptest.NewRequest("POST", "/register", strings.NewReader(`{"email": "john.doe@mail.com", "password": "p@$$w0rd"}`))
				r.Header.Set("Content-Type", "application/json;charset=utf-8")
				return r
			}(),
			opts: []testutil.HTTPOption{
				testutil.MaskRequestBody("password"),
				testutil.MaskResponseBody("data.accessToken"),
			},
			handler: loginHandler,
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
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

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

	bar, err := client.PostBar(ctx)
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

func (c *YourClient) PostBar(ctx context.Context) (string, error) {
	endpoint, err := url.JoinPath(c.url, "/bar")
	if err != nil {
		return "", err
	}

	fmt.Println("calling", endpoint)
	resp, err := http.Post(endpoint, "application/json", strings.NewReader(`{"bar": "baz"}`))
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
