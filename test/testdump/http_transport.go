package testdump

import (
	"net/http"

	"github.com/alextanhongpin/core/http/httputil"
)

type RoundTripper struct {
	RoundTripper http.RoundTripper
	opt          *HTTPOption
	hooks        []HTTPHook
	rw           readerWriter
}

func NewRoundTripper(rw readerWriter, opt *HTTPOption, hooks ...HTTPHook) *RoundTripper {
	return &RoundTripper{
		RoundTripper: http.DefaultTransport,
		opt:          opt,
		hooks:        hooks,
		rw:           rw,
	}
}

func (t *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r, err := httputil.CloneRequest(req)
	if err != nil {
		return nil, err
	}

	w, err := t.RoundTrip(req)

	if err := HTTP(t.rw, &HTTPDump{W: w, R: r}, t.opt, t.hooks...); err != nil {
		return w, err
	}

	return w, err
}
