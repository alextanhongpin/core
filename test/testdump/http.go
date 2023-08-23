package testdump

import (
	"fmt"
	"net/http"

	"github.com/alextanhongpin/core/http/httpdump"
	"github.com/alextanhongpin/core/internal"
	"github.com/google/go-cmp/cmp"
)

func HTTP(rw readerWriter, dump *HTTPDump, opt *HTTPOption) error {
	if opt == nil {
		opt = new(HTTPOption)
	}

	var s S[*HTTPDump] = &snapshot[*HTTPDump]{
		marshaler:   MarshalFunc[*HTTPDump](MarshalHTTP),
		unmarshaler: UnmarshalFunc[*HTTPDump](UnmarshalHTTP),
		comparer:    &HTTPComparer{opt: *opt},
	}

	return Snapshot(rw, dump, s, opt.Hooks...)
}

type HTTPDump struct {
	W *http.Response
	R *http.Request
}

func MarshalHTTP(d *HTTPDump) ([]byte, error) {
	return httpdump.DumpHTTP(d.W, d.R)
}

func UnmarshalHTTP(b []byte) (*HTTPDump, error) {
	w, r, err := httpdump.ReadHTTP(b)
	if err != nil {
		return nil, err
	}

	return &HTTPDump{
		W: w,
		R: r,
	}, nil
}

type HTTPHook = Hook[*HTTPDump]

type HTTPOption struct {
	Header  []cmp.Option
	Body    []cmp.Option
	Trailer []cmp.Option
	Hooks   []HTTPHook
}

type HTTPComparer struct {
	opt HTTPOption
}

func (c HTTPComparer) Compare(snapshot, received *HTTPDump) error {
	// Compare request.
	{
		snap, err := httpdump.FromRequest(snapshot.R)
		if err != nil {
			return err
		}
		recv, err := httpdump.FromRequest(received.R)
		if err != nil {
			return err
		}

		if err := compareHTTPDump(snap, recv, c.opt); err != nil {
			return fmt.Errorf("Request does not match snapshot. %w", err)
		}
	}

	// Compare response.
	{
		snap, err := httpdump.FromResponse(snapshot.W)
		if err != nil {
			return err
		}
		recv, err := httpdump.FromResponse(received.W)
		if err != nil {
			return err
		}

		if err := compareHTTPDump(snap, recv, c.opt); err != nil {
			return fmt.Errorf("Response does not match snapshot. %w", err)
		}
	}

	return nil
}

func compareHTTPDump(snapshot, received *httpdump.Dump, opt HTTPOption) error {
	x := snapshot
	y := received

	if err := internal.ANSIDiff(x.Line, y.Line); err != nil {
		return fmt.Errorf("Line: %w", err)
	}

	if err := internal.ANSIDiff(x.Body, y.Body, opt.Body...); err != nil {
		return fmt.Errorf("Body: %w", err)
	}

	if err := internal.ANSIDiff(x.Header, y.Header, opt.Header...); err != nil {
		return fmt.Errorf("Header: %w", err)
	}

	if err := internal.ANSIDiff(x.Trailer, y.Trailer, opt.Trailer...); err != nil {
		return fmt.Errorf("Trailer: %w", err)
	}

	return nil
}
