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

	if err := req.parse(); err != nil {
		logError(err)
		return nil, err
	}

	return req, nil
}

func (r *Request) parse() error {
	req, err := normalizeRequest(r.Request)
	if err != nil {
		logError(err)
		return err
	}

	dump, err := requestToDump(req)
	if err != nil {
		logError(err)
		return err
	}

	r.Request = req
	r.Dump = *dump

	return nil
}

func (r *Request) UnmarshalText(b []byte) error {
	b = normalizeNewlines(b)
	b = denormalizeNewlines(b)

	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(b)))
	if err != nil {
		logError(err)
		return err
	}

	req, err = normalizeRequest(req)
	if err != nil {
		logError(err)
		return err
	}

	dump, err := requestToDump(req)
	if err != nil {
		logError(err)
		return err
	}

	r.Request = req
	r.Dump = *dump

	return nil
}

func (r *Request) MarshalText() ([]byte, error) {
	// Use `DumpRequestOut` instead of `DumpRequest` to preserve the querystring.
	res, err := httputil.DumpRequestOut(r.Request, true)
	if err != nil {
		logError(err)
		return nil, err
	}

	res = normalizeNewlines(res)

	return res, nil
}

func normalizeRequest(r *http.Request) (*http.Request, error) {
	req := r.Clone(r.Context())

	// Prettify the request body.
	b, err := io.ReadAll(req.Body)
	if err != nil {
		logError(err)
		return nil, err
	}

	b, err = prettyBytes(b)
	if err != nil {
		logError(err)
		return nil, err
	}

	// NOTE: The new lines changes the content-length drastically.
	b = denormalizeNewlines(b)
	b = bytes.TrimSpace(b)
	req.Body = io.NopCloser(bytes.NewReader(b))

	// Update the content length.
	req.ContentLength = int64(len(b))

	// `httputil.DumpRequestOut` requires these to be set.
	normalizeHost(req)
	normalizeScheme(req)

	return req, nil
}

func normalizeHost(req *http.Request) {
	host := valueOrDefault(req.Header.Get("Host"), req.Host)
	host = valueOrDefault(host, "example.com")
	req.Header.Set("Host", host)
	req.Host = host
	req.URL.Host = host
}

func normalizeScheme(req *http.Request) {
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}
}

func requestToDump(req *http.Request) (*Dump, error) {
	reqLine := formatRequestLine(req)

	b, err := io.ReadAll(req.Body)
	if err != nil {
		logError(err)
		return nil, err
	}

	var a any
	if json.Valid(b) {
		if err := json.Unmarshal(b, &a); err != nil {
			return nil, err
		}
	} else {
		a = string(b)
	}

	req.Body = io.NopCloser(bytes.NewReader(b))

	return &Dump{
		Line:   reqLine,
		Header: req.Header.Clone(),
		Body:   a,
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
		logError(err)
		return nil, err
	}

	uri, err := url.Parse(reqURI)
	if err != nil {
		logError(err)
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
