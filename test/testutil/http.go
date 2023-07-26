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
)

type HTTPDumpOption = testdump.HTTPOption
type HTTPDump = testdump.HTTPDump
type HTTPHook = testdump.HTTPHook

type Path = internal.Path

type HTTPOption interface {
	isHTTP()
}

func DumpHTTPHandler(t *testing.T, r *http.Request, handler http.HandlerFunc, opts ...HTTPOption) {
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

	DumpHTTP(t, w, r, opts...)
}

func DumpHTTP(t *testing.T, w *http.Response, r *http.Request, opts ...HTTPOption) {
	t.Helper()

	o := new(httpOption)
	o.Dump = new(HTTPDumpOption)
	for _, opt := range opts {
		switch ot := opt.(type) {
		case FileName:
			o.FileName = string(ot)
		case httpOptionHook:
			ot(o)
		default:
			panic(fmt.Errorf("testutil: unhandled HTTP option: %#v", opt))
		}
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: o.FileName,
		FileExt:  ".http",
	}

	fileName := p.String()

	if err := testdump.HTTP(fileName, &HTTPDump{
		W: w,
		R: r,
	}, o.Dump); err != nil {
		t.Fatal(err)
	}
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

type httpOptionHook func(o *httpOption)

func (httpOptionHook) isHTTP() {}

type httpOption struct {
	Dump     *HTTPDumpOption
	FileName string
}

func IgnoreHeaders(headers ...string) httpOptionHook {
	return func(o *httpOption) {
		o.Dump.Header = append(o.Dump.Header,
			internal.IgnoreMapEntries(headers...),
		)
	}
}

func IgnoreTrailers(headers ...string) httpOptionHook {
	return func(o *httpOption) {
		o.Dump.Trailer = append(o.Dump.Trailer,
			internal.IgnoreMapEntries(headers...),
		)
	}
}

func IgnoreBodyFields(fields ...string) httpOptionHook {
	return func(o *httpOption) {
		o.Dump.Body = append(o.Dump.Body,
			internal.IgnoreMapEntries(fields...),
		)
	}
}

func MaskRequestHeaders(headers ...string) httpOptionHook {
	return func(o *httpOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MaskRequestHeaders(headers...),
		)
	}
}

func MaskResponseHeaders(headers ...string) httpOptionHook {
	return func(o *httpOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MaskResponseHeaders(headers...),
		)
	}
}

func MaskRequestBody(fields ...string) httpOptionHook {
	return func(o *httpOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MaskRequestBody(fields...),
		)
	}
}

func MaskResponseBody(fields ...string) httpOptionHook {
	return func(o *httpOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MaskResponseBody(fields...),
		)
	}
}

func InspectHTTP(hook func(snapshot, received *HTTPDump) error) httpOptionHook {
	return func(o *httpOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.CompareHook(hook))
	}
}

func InterceptHTTP(hook func(dump *HTTPDump) (*HTTPDump, error)) httpOptionHook {
	return func(o *httpOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MarshalHook(hook))
	}
}
