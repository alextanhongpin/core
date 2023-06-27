package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/httpdump"
	"github.com/google/go-cmp/cmp"
)

var (
	LineBreak = []byte("\n")
	Separator = []byte("###")
	SemiColon = []byte(":")
)

type httpOption struct {
	headerOpts   []cmp.Option
	bodyOpts     []cmp.Option
	interceptors []HTTPInterceptor
}

func NewHTTPOption(opts ...HTTPOption) *httpOption {
	h := &httpOption{}
	for _, opt := range opts {
		switch o := opt.(type) {
		case IgnoreFieldsOption:
			h.bodyOpts = append(h.bodyOpts, IgnoreMapKeys(o...))
		case IgnoreHeadersOption:
			h.headerOpts = append(h.headerOpts, IgnoreMapKeys(o...))
		case BodyCmpOptions:
			h.bodyOpts = append(h.bodyOpts, o...)
		case HeaderCmpOptions:
			h.headerOpts = append(h.headerOpts, o...)
		case HTTPInterceptor:
			h.interceptors = append(h.interceptors, o)
		case FilePath, FileName:
		default:
			panic("option not implemented")
		}
	}

	return h
}

func DumpHTTPFile(fileName string, r *http.Request, handler http.HandlerFunc, opts ...HTTPOption) error {
	// Make a copy of the body.
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	br := bytes.NewReader(b)
	r.Body = io.NopCloser(br)

	wr := httptest.NewRecorder()

	// Execute.
	handler(wr, r)
	w := wr.Result()

	// Reset request for the handler.
	br.Seek(0, 0)

	type dumpAndCompare struct {
		dumper
		comparer
	}

	opt := NewHTTPOption(opts...)
	for _, in := range opt.interceptors {
		if err := in(w, r); err != nil {
			return err
		}
	}

	dnc := &dumpAndCompare{
		dumper:   NewHTTPDumper(w, r),
		comparer: NewHTTPComparer(opts...),
	}

	return Dump(fileName, dnc)
}

func DumpHTTP(t *testing.T, r *http.Request, handler http.HandlerFunc, opts ...HTTPOption) string {
	t.Helper()
	p := NewHTTPPath(opts...)
	if p.FileName == "" {
		p.FileName = t.Name()
	}

	fileName := p.String()
	if err := DumpHTTPFile(fileName, r, handler, opts...); err != nil {
		t.Fatal(err)
	}

	return fileName
}

func DumpHTTPHandler(t *testing.T, opts ...HTTPOption) func(http.Handler) http.Handler {
	t.Helper()
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Serve to the response recorder.
			_ = DumpHTTP(t, r, next.ServeHTTP, opts...)

			// Serve to the actual server.
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

type HTTPDumper struct {
	w *http.Response
	r *http.Request
}

func NewHTTPDumper(w *http.Response, r *http.Request) *HTTPDumper {
	return &HTTPDumper{
		w: w,
		r: r,
	}
}

func (d *HTTPDumper) Dump() ([]byte, error) {
	req, err := httpdump.DumpRequest(d.r)
	if err != nil {
		return nil, err
	}

	res, err := httpdump.DumpResponse(d.w)
	if err != nil {
		return nil, err
	}

	return bytes.Join([][]byte{req, Separator, res}, bytes.Repeat(LineBreak, 2)), nil
}

type HTTPComparer struct {
	opt *httpOption
}

func NewHTTPComparer(opts ...HTTPOption) *HTTPComparer {
	return &HTTPComparer{
		opt: NewHTTPOption(opts...),
	}
}

func (c *HTTPComparer) Compare(want, got []byte) error {
	wantReq, wantRes, err := parseDotHTTP(want)
	if err != nil {
		return fmt.Errorf("failed to parse old snapshot: %w", err)
	}

	gotReq, gotRes, err := parseDotHTTP(got)
	if err != nil {
		return fmt.Errorf("failed to parse new snapshot: %w", err)
	}

	// httpDiff request.
	if err := httpDiff(wantReq, gotReq, c.opt.headerOpts, c.opt.bodyOpts); err != nil {
		return fmt.Errorf("Request does not match snapshot. %w", err)
	}

	// httpDiff response.
	if err := httpDiff(wantRes, gotRes, c.opt.headerOpts, c.opt.bodyOpts); err != nil {
		return fmt.Errorf("Response does not match snapshot. %w", err)
	}

	return nil
}

func parseDotHTTP(ss []byte) (reqS, resS *httpdump.Dump, err error) {
	var sep []byte
	sep = append(sep, LineBreak...)
	sep = append(sep, Separator...)
	sep = append(sep, LineBreak...)

	req, res, ok := bytes.Cut(ss, sep)
	if !ok {
		return nil, nil, fmt.Errorf("invalid snapshot: %s", ss)
	}

	reqS, err = httpdump.Parse(req)
	if err != nil {
		err = fmt.Errorf("failed to parse request: %w", err)
		return
	}

	resS, err = httpdump.Parse(res)
	if err != nil {
		err = fmt.Errorf("failed to parse response: %w", err)
		return
	}

	return
}

func httpDiff(x, y *httpdump.Dump, headerOpts []cmp.Option, bodyOpts []cmp.Option) error {
	if err := ansiDiff(x.Line, y.Line); err != nil {
		return err
	}

	// Compare body before header.
	// Headers may contain `Content-Length`, which depends on the body.
	if err := func(isJSON bool) error {
		if isJSON {
			comparer := &JSONComparer{opt: &jsonOption{bodyOpts: bodyOpts}}
			// Convert the json to map[string]any for better diff.
			// This does not work on JSON array.
			// Ensure that only structs are passed in.
			return comparer.Compare(x.Body, y.Body)
		}

		return ansiDiff(x.Body, y.Body, bodyOpts...)
	}(json.Valid(x.Body) && json.Valid(y.Body)); err != nil {
		return err
	}

	return ansiDiff(x.Headers, y.Headers, headerOpts...)
}
