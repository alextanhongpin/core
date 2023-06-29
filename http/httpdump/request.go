package httpdump

import (
	"bufio"
	"bytes"
	"encoding/json"
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

func NewRequest(r *http.Request) (*Request, error) {
	req := &Request{
		Request: r,
	}

	if err := req.Parse(); err != nil {
		return nil, err
	}

	return req, nil
}

func (r *Request) Parse() error {
	req, err := normalizeRequest(r.Request)
	if err != nil {
		return err
	}

	dump, err := requestToDump(req)
	if err != nil {
		return err
	}

	r.Request = req
	r.Dump = *dump

	return nil
}

func (r *Request) UnmarshalText(b []byte) error {
	b = bytes.TrimSpace(b)

	var req *http.Request

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
				var err error
				req, err = parseRequestLine(strings.NewReader(text))
				if err != nil {
					return err
				}

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

	req.Body = io.NopCloser(bytes.NewReader(bytes.TrimSpace(bb.Bytes())))

	var err error
	req, err = normalizeRequest(req)
	if err != nil {
		return err
	}

	dump, err := requestToDump(req)
	if err != nil {
		return err
	}

	r.Request = req
	r.Dump = *dump

	return nil
}

func (r *Request) MarshalText() ([]byte, error) {
	// Use `DumpRequestOut` instead of `DumpRequest` to preserve the
	// querystring.
	res, err := httputil.DumpRequestOut(r.Request, true)
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
	var dump Dump
	if err := json.Unmarshal(b, &dump); err != nil {
		return err
	}

	req, err := dumpToRequest(&dump)
	if err != nil {
		return err
	}

	r.Dump = dump
	r.Request = req

	return nil
}

func normalizeRequest(r *http.Request) (*http.Request, error) {
	req := r.Clone(r.Context())

	// Prettify the request body.
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	b, err = prettyBytes(b)
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(bytes.NewReader(b))

	// Update the content length.
	req.ContentLength = int64(len(b))

	// `httputil.DumpRequestOut` requires these to be set.
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}

	if req.URL.Host == "" {
		req.URL.Host = "example.com"
	}

	return req, nil
}

func dumpToRequest(dump *Dump) (*http.Request, error) {
	req, err := parseRequestLine(strings.NewReader(dump.Line))
	if err != nil {
		return nil, err
	}

	req.Header = dump.Header.Clone()
	req.Host = dump.Header.Get("Host")
	req.Body = io.NopCloser(dump.Body)

	return normalizeRequest(req)
}

func requestToDump(req *http.Request) (*Dump, error) {
	reqLine := formatRequestLine(req)

	b, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	body := bytes.NewReader(b)
	req.Body = io.NopCloser(body)

	return &Dump{
		Line:   reqLine,
		Header: req.Header.Clone(),
		Body:   body,
	}, nil
}

func formatRequestLine(req *http.Request) string {
	reqURI := req.RequestURI
	if reqURI == "" {
		reqURI = req.URL.RequestURI()
	}

	return fmt.Sprintf("%s %s HTTP/%d.%d", valueOrDefault(req.Method, "GET"),
		reqURI, req.ProtoMajor, req.ProtoMinor)
}

func parseRequestLine(r io.Reader) (*http.Request, error) {
	req := new(http.Request)

	var reqURI string
	if _, err := fmt.Fscanf(r, "%s %s HTTP/%d.%d",
		&req.Method,
		&reqURI,
		&req.ProtoMajor,
		&req.ProtoMinor,
	); err != nil {
		return nil, err
	}

	uri, err := url.Parse(reqURI)
	if err != nil {
		return nil, err
	}
	req.URL = uri
	req.Header = make(http.Header)

	return req, nil
}

func valueOrDefault(v, d string) string {
	if v != "" {
		return v
	}

	return d
}
