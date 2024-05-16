package testutil

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/httputil"
	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
	"github.com/google/go-cmp/cmp"
)

type HTTPDump = testdump.HTTPDump
type HTTPHook = testdump.HTTPHook

type Path = internal.Path

type HTTPOption interface {
	isHTTP()
}

func DumpHTTPHandler(t *testing.T, r *http.Request, handler http.HandlerFunc, opts ...HTTPOption) io.Reader {
	t.Helper()

	// Make a copy of the body.
	var reset func()
	//var br *bytes.Reader
	if r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		br := bytes.NewReader(b)
		r.Body = io.NopCloser(br)
		reset = func() {
			br.Seek(0, 0)
		}
	}

	wr := httptest.NewRecorder()

	// Execute.
	handler(wr, r)
	w := wr.Result()

	// Reset request for the handler.
	if reset != nil {
		reset()
	}

	return DumpHTTP(t, w, r, opts...)
}

func DumpHTTP(t *testing.T, w *http.Response, r *http.Request, opts ...HTTPOption) io.Reader {
	t.Helper()

	var fileName string
	var hooks []testdump.Hook[*HTTPDump]
	httpOpt := new(testdump.HTTPOption)

	for _, opt := range opts {
		switch o := opt.(type) {
		case FileName:
			fileName = string(o)
		case *httpHookOption:
			hooks = append(hooks, o.hook)
		case *httpCmpOption:
			httpOpt.Header = append(httpOpt.Header, o.header...)
			httpOpt.Body = append(httpOpt.Body, o.body...)
			httpOpt.Trailer = append(httpOpt.Trailer, o.trailer...)
		default:
			panic(fmt.Errorf("testutil: unhandled HTTP option: %#v", opt))
		}
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: fileName,
		FileExt:  ".http",
	}

	// Read the body and reset it.
	defer w.Body.Close()

	b, err := io.ReadAll(w.Body)
	if err != nil {
		t.Fatal(err)
	}

	br := bytes.NewReader(b)
	defer br.Seek(0, 0)
	w.Body = io.NopCloser(br)

	if err := testdump.HTTP(testdump.NewFile(p.String()), &HTTPDump{W: w, R: r}, httpOpt, hooks...); err != nil {
		t.Fatal(err)
	}

	return br
}

type RoundTripper struct {
	t    *testing.T
	opts []HTTPOption
}

func DumpRoundTrip(t *testing.T, opts ...HTTPOption) *RoundTripper {
	return &RoundTripper{
		t:    t,
		opts: opts,
	}
}

func (t *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r, err := httputil.CloneRequest(req)
	if err != nil {
		return nil, err
	}

	w, err := http.DefaultTransport.RoundTrip(req)

	DumpHTTP(t.t, w, r, t.opts...)

	return w, err
}

type httpHookOption struct {
	hook testdump.Hook[*HTTPDump]
}

func (httpHookOption) isHTTP() {}

type httpCmpOption struct {
	header  []cmp.Option
	body    []cmp.Option
	trailer []cmp.Option
}

func (httpCmpOption) isHTTP() {}

func IgnoreHeaders(headers ...string) *httpCmpOption {
	return &httpCmpOption{
		header: []cmp.Option{internal.IgnoreMapEntries(headers...)},
	}
}

func IgnoreTrailers(headers ...string) *httpCmpOption {
	return &httpCmpOption{
		trailer: []cmp.Option{internal.IgnoreMapEntries(headers...)},
	}
}

func IgnoreBodyFields(fields ...string) *httpCmpOption {
	return &httpCmpOption{
		body: []cmp.Option{internal.IgnoreMapEntries(fields...)},
	}
}

func MaskRequestHeaders(headers ...string) *httpHookOption {
	return &httpHookOption{
		hook: testdump.MaskRequestHeaders(headers...),
	}
}

func MaskResponseHeaders(headers ...string) *httpHookOption {
	return &httpHookOption{
		hook: testdump.MaskResponseHeaders(headers...),
	}
}

func MaskRequestBody(fields ...string) *httpHookOption {
	return &httpHookOption{
		hook: testdump.MaskRequestBody(fields...),
	}
}

func MaskResponseBody(fields ...string) *httpHookOption {
	return &httpHookOption{
		hook: testdump.MaskResponseBody(fields...),
	}
}

func InspectHTTP(hook func(snapshot, received *HTTPDump) error) *httpHookOption {
	return &httpHookOption{
		hook: testdump.CompareHook(hook),
	}
}

func InterceptHTTP(hook func(dump *HTTPDump) (*HTTPDump, error)) *httpHookOption {
	return &httpHookOption{
		hook: testdump.MarshalHook(hook),
	}
}
