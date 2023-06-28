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
	*http.Request
	Dump Dump
}

func NewRequest(r *http.Request) *Request {
	return &Request{
		Request: r,
	}
}

func (r *Request) UnmarshalBinary(b []byte) error {
	b = bytes.TrimSpace(b)

	req := new(http.Request)
	req.Header = make(http.Header)

	scanner := bufio.NewScanner(bytes.NewReader(b))
	var bb bytes.Buffer

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
				var reqURI string
				if _, err := fmt.Fscanf(
					bytes.NewReader(b),
					"%s %s HTTP/%d.%d\r\n",
					&req.Method,
					&reqURI,
					&req.ProtoMajor,
					&req.ProtoMinor,
				); err != nil {
					return err
				}

				uri, err := url.Parse(reqURI)
				if err != nil {
					return err
				}
				req.URL = uri

				r.Dump.Line = text

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

	body := bytes.NewReader(bytes.TrimSpace(bb.Bytes()))
	req.Body = io.NopCloser(body)

	r.Dump.Header = req.Header.Clone()
	r.Dump.Body = body

	r.Request = new(http.Request)
	*r.Request = *req

	return nil
}

func (r *Request) MarshalBinary() ([]byte, error) {
	req := r.Request
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	b, err = prettyBytes(b)
	if err != nil {
		return nil, err
	}

	// Assign back to the body.
	req.Body = io.NopCloser(bytes.NewReader(b))

	// Update the content-length after updating body.
	req.ContentLength = int64(len(b))

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
