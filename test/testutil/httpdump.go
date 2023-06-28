package testutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"text/template"

	"github.com/alextanhongpin/core/http/httpdump"
	"github.com/google/go-cmp/cmp"
)

var ErrInvalidHTTPDumpFormat = errors.New("invalid HTTP dump format")

var (
	LineBreak = []byte("\n")

	dumpTemplate = template.Must(template.New(`request`).Parse(`{{.Request}}


###


{{.Response}}
`))

	re = regexp.MustCompile(`(?m)^#{3}$`)
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
	r := httpdump.NewRequest(d.r)
	w := httpdump.NewResponse(d.w)

	req, err := r.MarshalBinary()
	if err != nil {
		return nil, err
	}

	res, err := w.MarshalBinary()
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := dumpTemplate.Execute(&b, map[string]any{
		"Request":  string(req),
		"Response": string(res),
	}); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
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
	snapshotReq, snapshotRes, err := parseHTTPDump(snapshot)
	if err != nil {
		return fmt.Errorf("failed to parse old snapshot: %w", err)
	}

	receivedReq, receivedRes, err := parseHTTPDump(received)
	if err != nil {
		return fmt.Errorf("failed to parse new snapshot: %w", err)
	}

	// httpdumpDiff request.
	if err := httpdumpDiff(snapshotReq.Dump, receivedReq.Dump, c.opt.headerOpts, c.opt.bodyOpts); err != nil {
		return fmt.Errorf("Request does not match snapshot. %w", err)
	}

	// httpdumpDiff response.
	if err := httpdumpDiff(snapshotRes.Dump, receivedRes.Dump, c.opt.headerOpts, c.opt.bodyOpts); err != nil {
		return fmt.Errorf("Response does not match snapshot. %w", err)
	}

	return nil
}

func parseHTTPDump(b []byte) (r *httpdump.Request, w *httpdump.Response, err error) {
	req, res, err := split(b)
	if err != nil {
		return nil, nil, err
	}

	r = new(httpdump.Request)
	err = r.UnmarshalBinary(req)
	if err != nil {
		return
	}

	w = new(httpdump.Response)
	err = w.UnmarshalBinary(res)
	if err != nil {
		return
	}

	return
}

type dump struct {
	Line   string
	Header http.Header
	Body   *bytes.Buffer
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

	xBody, err := io.ReadAll(x.Body)
	if err != nil {
		return err
	}
	x.Body.Seek(0, 0)

	yBody, err := io.ReadAll(y.Body)
	if err != nil {
		return err
	}
	y.Body.Seek(0, 0)

	// Compare body before header.
	// Headers may contain `Content-Length`, which depends on the body.
	if err := func(isJSON bool) error {
		if isJSON {
			comparer := &JSONComparer{opt: &jsonOption{bodyOpts: bodyOpts}}
			// Convert the json to map[string]any for better diff.
			// This does not work on JSON array.
			// Ensure that only structs are passed in.
			return comparer.Compare(xBody, yBody)
		}

		return ansiDiff(xBody, yBody, bodyOpts...)
	}(json.Valid(xBody) && json.Valid(yBody)); err != nil {
		return err
	}

	return ansiDiff(x.Header, y.Header, headerOpts...)
}

func split(s []byte) (req, res []byte, err error) {
	parts := re.Split(string(s), 2)
	if len(parts) != 2 {
		return nil, nil, ErrInvalidHTTPDumpFormat
	}

	req = []byte(parts[0])
	res = []byte(parts[1])

	return
}
