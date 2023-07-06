package testutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/httpdump"
	"github.com/google/go-cmp/cmp"
)

var ErrInvalidHTTPDumpFormat = errors.New("invalid HTTP dump format")

var LineBreak = []byte("\n")

type httpOption struct {
	headerOpts   []cmp.Option
	bodyOpts     []cmp.Option
	interceptors []HTTPInterceptor
}

func NewHTTPOption(opts ...HTTPOption) *httpOption {
	h := &httpOption{}
	for _, opt := range opts {
		switch o := opt.(type) {
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

func DumpHTTPHandler(t *testing.T, r *http.Request, handler http.HandlerFunc, opts ...HTTPOption) string {
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

func DumpHTTP(t *testing.T, w *http.Response, r *http.Request, opts ...HTTPOption) string {
	t.Helper()

	type dumpAndCompare struct {
		dumper
		comparer
	}

	opt := NewHTTPOption(opts...)
	for _, in := range opt.interceptors {
		if err := in(w, r); err != nil {
			t.Fatal(err)
		}
	}

	dnc := &dumpAndCompare{
		dumper:   NewHTTPDumper(w, r),
		comparer: NewHTTPComparer(opts...),
	}

	p := NewHTTPPath(opts...)
	if p.FileName == "" {
		p.FileName = t.Name()
	}

	fileName := p.String()
	if err := Dump(fileName, dnc); err != nil {
		t.Fatal(err)
	}

	return fileName
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
	return httpdump.DumpHTTP(d.w, d.r)
}

type HTTPComparer struct {
	opt *httpOption
}

func NewHTTPComparer(opts ...HTTPOption) *HTTPComparer {
	return &HTTPComparer{
		opt: NewHTTPOption(opts...),
	}
}

func (c *HTTPComparer) Compare(snapshot, received []byte) error {
	w, r, err := httpdump.ReadHTTP(snapshot)
	if err != nil {
		return err
	}
	xr, err := httpdump.FromRequest(r)
	if err != nil {
		return nil
	}
	xw, err := httpdump.FromResponse(w)
	if err != nil {
		return nil
	}

	ww, rr, err := httpdump.ReadHTTP(received)
	if err != nil {
		return err
	}

	yr, err := httpdump.FromRequest(rr)
	if err != nil {
		return nil
	}

	yw, err := httpdump.FromResponse(ww)
	if err != nil {
		return nil
	}

	if err := httpdumpDiff(*xr, *yr, c.opt.headerOpts, c.opt.bodyOpts); err != nil {
		return fmt.Errorf("Request does not match snapshot. %w", err)
	}

	if err := httpdumpDiff(*xw, *yw, c.opt.headerOpts, c.opt.bodyOpts); err != nil {
		return fmt.Errorf("Response does not match snapshot. %w", err)
	}

	return nil
}

func httpdumpDiff(
	snapshot httpdump.Dump,
	received httpdump.Dump,
	headerOpts []cmp.Option,
	bodyOpts []cmp.Option,
) error {
	x := snapshot
	y := received

	if err := ansiDiff(x.Line, y.Line); err != nil {
		return fmt.Errorf("Line: %w", err)
	}

	if err := ansiDiff(x.Body, y.Body, bodyOpts...); err != nil {
		return fmt.Errorf("Body: %w", err)
	}

	if err := ansiDiff(x.Header, y.Header, headerOpts...); err != nil {
		return fmt.Errorf("Header: %w", err)
	}

	if err := ansiDiff(x.Trailer, y.Trailer, headerOpts...); err != nil {
		return fmt.Errorf("Trailer: %w", err)
	}

	return nil
}
