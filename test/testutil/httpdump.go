package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var (
	LineBreak = []byte("\n")
	Separator = []byte("---")
	SemiColon = []byte(":")
)

type Options struct {
	headopts []cmp.Option
	bodyopts []cmp.Option
	bodyFn   func(body []byte)
	headerFn func(http.Header)
}

type Option func(*Options)

func ignoreMapKeys(keys ...string) cmp.Option {
	return cmpopts.IgnoreMapEntries(func(key string, _ any) bool {
		for _, k := range keys {
			if k == key {
				return true
			}
		}

		return false
	})
}

func IgnoreHeaders(keys ...string) Option {
	return func(o *Options) {
		o.headopts = append(o.headopts, ignoreMapKeys(keys...))
	}
}

func IgnoreFields(keys ...string) Option {
	return func(o *Options) {
		o.bodyopts = append(o.bodyopts, ignoreMapKeys(keys...))
	}
}

func InspectBody(fn func(body []byte)) Option {
	return func(o *Options) {
		o.bodyFn = fn
	}
}

func InspectHeaders(fn func(http.Header)) Option {
	return func(o *Options) {
		o.headerFn = fn
	}
}

func DumpHTTP(t *testing.T, r *http.Request, handler http.HandlerFunc, opts ...Option) {
	t.Helper()

	// Execute.
	wr := httptest.NewRecorder()
	handler(wr, r)

	w := wr.Result()

	// Prepare dump.
	req, err := dumpRequest(r)
	if err != nil {
		t.Fatal(err)
	}

	res, err := dumpResponse(w)
	if err != nil {
		t.Fatal(err)
	}

	// Dump in this format.
	got := bytes.Join([][]byte{req, Separator, res}, bytes.Repeat(LineBreak, 2))

	// Save in testdata directory
	// Save as .http files.
	// Skip if file exists.
	fileName := fmt.Sprintf("./testdata/%s.http", t.Name())
	if err := writeToNewFile(fileName, got); err != nil {
		t.Fatal(err)
	}

	// Read existing snapshot.
	want, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatal(err)
	}

	// Compare the new snapshot with existing snapshot.
	if err := compareSnapshot(want, got, opts...); err != nil {
		t.Fatal(err)
	}
}

func compareSnapshot(want, got []byte, opts ...Option) error {
	if bytes.Equal(want, got) {
		return nil
	}

	o := new(Options)
	for _, opt := range opts {
		opt(o)
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
	if err := wantReq.Diff(gotReq, o); err != nil {
		return fmt.Errorf("Request does not match snapshot. %w", err)
	}

	// Diff response.
	if err := wantRes.Diff(gotRes, o); err != nil {
		return fmt.Errorf("Response does not match snapshot. %w", err)
	}

	// Validate response body.
	if o.bodyFn != nil {
		o.bodyFn(wantRes.Body)
	}

	if o.headerFn != nil {
		o.headerFn(wantRes.Headers)
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

func (s *Snapshot) Diff(other *Snapshot, opts *Options) error {
	if diff := cmp.Diff(s.Line, other.Line); diff != "" {
		return diffError(diff)
	}

	// Compare body before header.
	// Headers may contain `Content-Length`, which depends on the body.
	if json.Valid(s.Body) && json.Valid(other.Body) {
		// Convert the json to map[string]any for better diff.
		if err := cmpJSON(s.Body, other.Body, opts.bodyopts...); err != nil {
			return err
		}
	} else {
		if diff := cmp.Diff(s.Body, other.Body, opts.bodyopts...); diff != "" {
			return diffError(diff)
		}
	}

	diff := cmp.Diff(s.Headers, other.Headers, opts.headopts...)
	return diffError(diff)
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
