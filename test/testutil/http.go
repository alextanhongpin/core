package testutil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
)

// IgnoreHeaders
// MaskRequestBody
// MaskResponseBody
// MaskResponseHeaders
// MaskRequestHeaders
type HTTPDumpOption = testdump.HTTPOption
type HTTPDump = testdump.HTTPDump
type HTTPHook = testdump.HTTPHook

type Path = internal.Path

type HTTPOption struct {
	Dump     *HTTPDumpOption
	FileName string
}

func DumpHTTPHandler(t *testing.T, r *http.Request, handler http.HandlerFunc, opt *HTTPOption) {
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

	DumpHTTP(t, w, r, opt)
}

func DumpHTTP(t *testing.T, w *http.Response, r *http.Request, opt *HTTPOption) {
	t.Helper()

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: opt.FileName,
		FileExt:  ".http",
	}

	fileName := p.String()

	if err := testdump.HTTP(fileName, &testdump.HTTPDump{
		W: w,
		R: r,
	}, opt.Dump); err != nil {
		t.Fatal(err)
	}
}
