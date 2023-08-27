package testdump

import (
	"fmt"
	"net/http"

	"github.com/alextanhongpin/core/http/httpdump"
	"github.com/alextanhongpin/core/internal"
	"github.com/google/go-cmp/cmp"
)

type HTTPDump struct {
	W *http.Response
	R *http.Request
}

type HTTPHook = Hook[*HTTPDump]

type HTTPOption struct {
	Header  []cmp.Option
	Body    []cmp.Option
	Trailer []cmp.Option
}

func HTTP(rw readerWriter, dump *HTTPDump, opt *HTTPOption, hooks ...HTTPHook) error {
	if opt == nil {
		opt = new(HTTPOption)
	}

	var s S[*HTTPDump] = &snapshot[*HTTPDump]{
		marshaler:   MarshalFunc[*HTTPDump](MarshalHTTP),
		unmarshaler: UnmarshalFunc[*HTTPDump](UnmarshalHTTP),
		comparer: &HTTPComparer{
			Header:  opt.Header,
			Body:    opt.Body,
			Trailer: opt.Trailer,
		},
	}

	s = Hooks[*HTTPDump](hooks).Apply(s)

	return Snapshot(rw, dump, s)
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

type HTTPComparer struct {
	Header  []cmp.Option
	Body    []cmp.Option
	Trailer []cmp.Option
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

		if err := c.compare(snap, recv); err != nil {
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

		if err := c.compare(snap, recv); err != nil {
			return fmt.Errorf("Response does not match snapshot. %w", err)
		}
	}

	return nil
}

func (c *HTTPComparer) compare(snapshot, received *httpdump.Dump) error {
	x := snapshot
	y := received

	if err := internal.ANSIDiff(x.Line, y.Line); err != nil {
		return fmt.Errorf("Line: %w", err)
	}

	if err := internal.ANSIDiff(x.Body, y.Body, c.Body...); err != nil {
		return fmt.Errorf("Body: %w", err)
	}

	if err := internal.ANSIDiff(x.Header, y.Header, c.Header...); err != nil {
		return fmt.Errorf("Header: %w", err)
	}

	if err := internal.ANSIDiff(x.Trailer, y.Trailer, c.Trailer...); err != nil {
		return fmt.Errorf("Trailer: %w", err)
	}

	return nil
}
