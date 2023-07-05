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
	h, err := httpdump.NewHTTP(d.w, d.r)
	if err != nil {
		return nil, err
	}

	return h.MarshalText()
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
	snap := new(httpdump.HTTP)
	if err := snap.UnmarshalText(snapshot); err != nil {
		return err
	}

	recv := new(httpdump.HTTP)
	if err := recv.UnmarshalText(snapshot); err != nil {
		return err
	}

	if err := httpdumpDiff(
		snap.Request().Dump,
		recv.Request().Dump,
		c.opt.headerOpts,
		c.opt.bodyOpts,
	); err != nil {
		return fmt.Errorf("Request does not match snapshot. %w", err)
	}

	if err := httpdumpDiff(
		snap.Response().Dump,
		recv.Response().Dump,
		c.opt.headerOpts,
		c.opt.bodyOpts,
	); err != nil {
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
		return err
	}

	if err := ansiDiff(x.Body, y.Body, bodyOpts...); err != nil {
		return err
	}

	return ansiDiff(x.Header, y.Header, headerOpts...)
}
