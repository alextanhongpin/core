package testdump

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/alextanhongpin/core/http/httputil"
)

type readerWriterFunc func(w *http.Response, r *http.Request) readerWriter

// HTTPFile returns a filename based on the http request method and response
// status code.
func HTTPFile(w *http.Response, r *http.Request) readerWriter {
	file := fmt.Sprintf("%d.http", w.StatusCode)
	name := filepath.Join(r.Method, r.URL.Path, file)
	return NewFile(name)
}

type RoundTripper struct {
	RoundTripper http.RoundTripper
	opt          *HTTPOption
	hooks        []HTTPHook
	rwFunc       readerWriterFunc
}

func NewRoundTripper(rwFunc readerWriterFunc, opt *HTTPOption, hooks ...HTTPHook) *RoundTripper {
	return &RoundTripper{
		RoundTripper: http.DefaultTransport,
		opt:          opt,
		hooks:        hooks,
		rwFunc:       rwFunc,
	}
}

func (t *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r, err := httputil.CloneRequest(req)
	if err != nil {
		return nil, err
	}

	w, err := t.RoundTrip(req)

	if err := HTTP(t.rwFunc(w, r), &HTTPDump{W: w, R: r}, t.opt, t.hooks...); err != nil {
		return w, err
	}

	return w, err
}
