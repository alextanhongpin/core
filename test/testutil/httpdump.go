package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	LineBreak = []byte("\n")
	Separator = []byte("---")
	SemiColon = []byte(":")
)

type httpOption struct {
	headerFn   func(http.Header)
	headerOpts []cmp.Option
	bodyFn     func([]byte)
	bodyOpts   []cmp.Option
}

func NewHTTPOption(opts ...HTTPOption) *httpOption {
	h := &httpOption{}
	for _, opt := range opts {
		switch o := opt.(type) {
		case *InspectBodyOption:
			h.bodyFn = o.fn
		case *InspectHeadersOption:
			h.headerFn = o.fn
		case *IgnoreFieldsOption:
			h.bodyOpts = append(h.bodyOpts, ignoreMapKeys(o.keys...))
		case *IgnoreHeadersOption:
			h.headerOpts = append(h.headerOpts, ignoreMapKeys(o.keys...))
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
	defer br.Seek(0, 0)

	r.Body.Close()
	r.Body = io.NopCloser(br)

	wr := httptest.NewRecorder()

	// Execute.
	handler(wr, r)
	w := wr.Result()

	// Restore to original body.
	br.Seek(0, 0)

	type dumpAndCompare struct {
		dumper
		comparer
	}

	dnc := &dumpAndCompare{
		dumper:   NewHTTPDumper(w, r),
		comparer: NewHTTPComparer(opts...),
	}

	return Dump(fileName, dnc)
}

func DumpHTTP(t *testing.T, r *http.Request, handler http.HandlerFunc, opts ...HTTPOption) {
	t.Helper()

	fileName := fmt.Sprintf("./testdata/%s.http", t.Name())
	if err := DumpHTTPFile(fileName, r, handler, opts...); err != nil {
		t.Fatal(err)
	}
}

func DumpHTTPHandler(t *testing.T, opts ...HTTPOption) func(http.Handler) http.Handler {
	t.Helper()
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Serve to the response recorder.
			DumpHTTP(t, r, next.ServeHTTP, opts...)

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
	return &HTTPDumper{w, r}
}

func (d *HTTPDumper) Dump() ([]byte, error) {
	req, err := dumpRequest(d.r)
	if err != nil {
		return nil, err
	}

	res, err := dumpResponse(d.w)
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
	if bytes.Equal(want, got) {
		return nil
	}

	wantReq, wantRes, err := parseDotHTTP(want)
	if err != nil {
		return fmt.Errorf("failed to parse old snapshot: %w", err)
	}

	gotReq, gotRes, err := parseDotHTTP(got)
	if err != nil {
		return fmt.Errorf("failed to parse new snapshot: %w", err)
	}

	// Diff request.
	if err := wantReq.Diff(gotReq, c.opt.headerOpts, c.opt.bodyOpts); err != nil {
		return fmt.Errorf("Request does not match snapshot. %w", err)
	}

	// Diff response.
	if err := wantRes.Diff(gotRes, c.opt.headerOpts, c.opt.bodyOpts); err != nil {
		return fmt.Errorf("Response does not match snapshot. %w", err)
	}

	// Validate response body.
	if c.opt.bodyFn != nil {
		c.opt.bodyFn(wantRes.Body)
	}

	if c.opt.headerFn != nil {
		c.opt.headerFn(wantRes.Headers)
	}

	return nil
}

func parseDotHTTP(ss []byte) (reqS, resS *Snapshot, err error) {
	req, res, ok := bytes.Cut(ss, Separator)
	if !ok {
		return nil, nil, fmt.Errorf("invalid snapshot: %s", ss)
	}

	reqS, err = parseSection(req)
	if err != nil {
		err = fmt.Errorf("failed to parse request: %w", err)
		return
	}

	resS, err = parseSection(res)
	if err != nil {
		err = fmt.Errorf("failed to parse response: %w", err)
		return
	}

	return
}

type Snapshot struct {
	Line    string
	Headers http.Header
	Body    []byte
}

func (s *Snapshot) Diff(other *Snapshot, headerOpts []cmp.Option, bodyOpts []cmp.Option) error {
	if err := cmpDiff(s.Line, other.Line); err != nil {
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
			return comparer.Compare(s.Body, other.Body)
		}

		return cmpDiff(s.Body, other.Body, bodyOpts...)
	}(json.Valid(s.Body) && json.Valid(other.Body)); err != nil {
		return err
	}

	return cmpDiff(s.Headers, other.Headers, headerOpts...)
}

func parseSection(req []byte) (*Snapshot, error) {
	req = bytes.TrimSpace(req)
	rawReqLine, rawHeadersAndBody, _ := bytes.Cut(req, LineBreak)
	rawHeaders, body, _ := bytes.Cut(rawHeadersAndBody, bytes.Repeat(LineBreak, 2))
	headers, err := parseHeaders(rawHeaders)
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		Line:    string(bytes.TrimSpace(rawReqLine)),
		Headers: headers,
		Body:    bytes.TrimSpace(body),
	}, nil
}

// parseHeaders parse the HTTP headers from key-value strings into map of
// strings.
func parseHeaders(headers []byte) (http.Header, error) {
	headers = bytes.TrimSpace(headers)

	h := make(http.Header)
	kvs := bytes.Split(headers, LineBreak)
	for _, kv := range kvs {
		k, v, ok := bytes.Cut(kv, SemiColon)
		if !ok {
			return nil, fmt.Errorf("invalid header format: %q", kv)
		}
		ks := string(bytes.TrimSpace(k))
		vs := string(bytes.TrimSpace(v))
		h[ks] = append(h[ks], vs)
	}

	return h, nil
}

func dumpRequest(r *http.Request) ([]byte, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var br interface {
		io.Reader
		Len() int
	}

	if json.Valid(b) {
		// Pretty-print the json body.
		bb := new(bytes.Buffer)
		if err := json.Indent(bb, b, "", " "); err != nil {
			return nil, err
		}
		br = bb
	} else {
		br = bytes.NewReader(b)
	}
	// Assign back to the body.
	r.Body = io.NopCloser(br)

	// Update the content-length after updating body.
	r.ContentLength = int64(br.Len())

	// `httputil.DumpRequestOut` requires these to be set.
	if r.URL.Scheme == "" {
		r.URL.Scheme = "http"
	}
	if r.URL.Host == "" {
		r.URL.Host = "example.com"
	}

	// Use `DumpRequestOut` instead of `DumpRequest` to preserve the
	// querystring.
	req, err := httputil.DumpRequestOut(r, true)
	if err != nil {
		return nil, err
	}
	req = NormalizeNewlines(req)
	req = bytes.TrimSpace(req)

	return req, nil
}

func dumpResponse(w *http.Response) ([]byte, error) {
	b, err := io.ReadAll(w.Body)
	if err != nil {
		return nil, err
	}
	if json.Valid(b) {
		// Pretty-print the json body.
		bb := new(bytes.Buffer)
		if err := json.Indent(bb, b, "", " "); err != nil {
			return nil, err
		}
		// Assign back to the body.
		w.Body = io.NopCloser(bb)
	} else {
		// Assign back to the body.
		w.Body = io.NopCloser(bytes.NewReader(b))
	}

	res, err := httputil.DumpResponse(w, true)
	if err != nil {
		return nil, err
	}

	res = NormalizeNewlines(res)
	res = bytes.TrimSpace(res)

	return res, nil
}

// NormalizeNewlines normalizes \r\n (windows) and \r (mac)
// into \n (unix)
// Reference [here].
// [here]: https://www.programming-books.io/essential/go/normalize-newlines-1d3abcf6f17c4186bb9617fa14074e48
func NormalizeNewlines(d []byte) []byte {
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = bytes.Replace(d, []byte{13, 10}, []byte{10}, -1)
	// replace CF \r (mac) with LF \n (unix)
	d = bytes.Replace(d, []byte{13}, []byte{10}, -1)
	return d
}
