package httpdump

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

var ErrParseHeader = errors.New("httpdump: parse header failed")

type Request struct {
	*http.Request `json:"-"`
	Dump          Dump
}

func NewRequest(r *http.Request) *Request {
	return &Request{
		Request: r,
	}
}

func (r *Request) Parse() error {
	b, err := r.MarshalText()
	if err != nil {
		return err
	}

	return r.UnmarshalText(b)
}

func (r *Request) UnmarshalText(b []byte) error {
	b = bytes.TrimSpace(b)

	var req *http.Request

	scanner := bufio.NewScanner(bytes.NewReader(b))

	var bb bytes.Buffer
	var line string

	sections := 3
	for i := 0; i < sections; i++ {
	scan:
		for scanner.Scan() {
			text := scanner.Text()
			if len(text) == 0 {
				break scan
			}

			switch i {
			case 0:
				var err error
				req, err = parseLineRequest([]byte(text))
				if err != nil {
					return err
				}
				line = text

				break scan
			case 1:
				k, v, ok := strings.Cut(text, ": ")
				if !ok {
					return fmt.Errorf("%w: %q", ErrParseHeader, text)
				}
				req.Header.Add(k, v)
				if http.CanonicalHeaderKey(k) == "Host" {
					req.Host = v
				}
			case 2:
				bb.WriteString(text)
				bb.WriteString("\n")
			}
		}
	}

	b = bytes.TrimSpace(bb.Bytes())
	var err error
	b, err = prettyBytes(b)
	if err != nil {
		return err
	}

	body := bytes.NewReader(b)
	req.Body = io.NopCloser(body)

	req.ContentLength = int64(len(b))

	r.Dump = Dump{
		Line:   line,
		Header: req.Header.Clone(),
		Body:   body,
	}

	r.Request = new(http.Request)
	r.Request.Header = make(http.Header)

	*r.Request = *req

	return nil
}

func (r *Request) MarshalText() ([]byte, error) {
	req := r.Request

	// Prettify the request body.
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	b, err = prettyBytes(b)
	if err != nil {
		return nil, err
	}

	// Update the content length.
	req.ContentLength = int64(len(b))

	req.Body = io.NopCloser(bytes.NewReader(b))

	// `httputil.DumpRequestOut` requires these to be set.
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}

	if req.URL.Host == "" {
		req.URL.Host = "example.com"
	}

	// Use `DumpRequestOut` instead of `DumpRequest` to preserve the
	// querystring.
	res, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, err
	}
	res = NormalizeNewlines(res)
	res = bytes.TrimSpace(res)

	return res, nil
}

func (r *Request) MarshalJSON() ([]byte, error) {
	return r.Dump.MarshalJSON()
}

func (r *Request) UnmarshalJSON(b []byte) error {
	dump := new(Dump)
	dump.Header = make(http.Header)
	if err := dump.UnmarshalJSON(b); err != nil {
		return err
	}
	r.Dump = *dump

	req, err := parseLineRequest([]byte(dump.Line))
	if err != nil {
		return err
	}

	req.Header = dump.Header.Clone()
	req.Host = dump.Header.Get("Host")
	req.Body = io.NopCloser(dump.Body)

	r.Request = new(http.Request)
	*r.Request = *req

	return err
}

func parseLineRequest(b []byte) (*http.Request, error) {
	r := new(http.Request)

	var reqURI string
	if _, err := fmt.Fscanf(
		bytes.NewReader(b),
		"%s %s HTTP/%d.%d",
		&r.Method,
		&reqURI,
		&r.ProtoMajor,
		&r.ProtoMinor,
	); err != nil {
		return nil, err
	}

	uri, err := url.Parse(reqURI)
	if err != nil {
		return nil, err
	}
	r.URL = uri
	r.Header = make(http.Header)

	return r, nil
}
