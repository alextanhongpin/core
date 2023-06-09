package testutil

import (
	"bytes"
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

type HTTPOption func(o *HttpOption)

type HttpOption struct {
	Dump     *HTTPDumpOption
	FileName string
}

func DumpHTTPHandler(t *testing.T, r *http.Request, handler http.HandlerFunc, opts ...HTTPOption) {
	t.Helper()

	// Make a copy of the body.
	b, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}

	br := bytes.NewReader(b)
	r.Body = io.NopCloser(br)

	wr := httptest.NewRecorder()

	// Execute.
	handler(wr, r)
	w := wr.Result()

	// Reset request for the handler.
	br.Seek(0, 0)

	DumpHTTP(t, w, r, opts...)
}

func DumpHTTP(t *testing.T, w *http.Response, r *http.Request, opts ...HTTPOption) {
	t.Helper()

	o := new(HttpOption)
	o.Dump = new(HTTPDumpOption)
	for _, opt := range opts {
		opt(o)
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

func IgnoreHeaders(headers ...string) HTTPOption {
	return func(o *HttpOption) {
		o.Dump.Header = append(o.Dump.Header,
			internal.IgnoreMapEntries(headers...),
		)
	}
}
func IgnoreTrailers(headers ...string) HTTPOption {
	return func(o *HttpOption) {
		o.Dump.Trailer = append(o.Dump.Trailer,
			internal.IgnoreMapEntries(headers...),
		)
	}
}
func IgnoreBodyFields(fields ...string) HTTPOption {
	return func(o *HttpOption) {
		o.Dump.Body = append(o.Dump.Body,
			internal.IgnoreMapEntries(fields...),
		)
	}
}

func MaskRequestHeaders(headers ...string) HTTPOption {
	return func(o *HttpOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MaskRequestHeaders(headers...),
		)
	}
}

func MaskResponseHeaders(headers ...string) HTTPOption {
	return func(o *HttpOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MaskResponseHeaders(headers...),
		)
	}
}

func MaskRequestBody(fields ...string) HTTPOption {
	return func(o *HttpOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MaskRequestBody(fields...),
		)
	}
}

func MaskResponseBody(fields ...string) HTTPOption {
	return func(o *HttpOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MaskResponseBody(fields...),
		)
	}
}

func InspectHTTP(hook func(snapshot, received *HTTPDump) error) HTTPOption {
	return func(o *HttpOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.CompareHook(hook))
	}
}

func InterceptHTTP(hook func(dump *HTTPDump) (*HTTPDump, error)) HTTPOption {
	return func(o *HttpOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MarshalHook(hook))
	}
}

func HTTPFileName(name string) HTTPOption {
	return func(o *HttpOption) {
		o.FileName = name
	}
}
