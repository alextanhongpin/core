package httpdump

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
)

var (
	SemiColon = []byte(":")
	LineBreak = []byte("\n")
)

func DumpRequest(r *http.Request) ([]byte, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	b, err = prettyBytes(b)
	if err != nil {
		return nil, err
	}

	// Assign back to the body.
	r.Body = io.NopCloser(bytes.NewReader(b))

	// Update the content-length after updating body.
	r.ContentLength = int64(len(b))

	// `httputil.DumpRequestOut` requires these to be set.
	if r.URL.Scheme == "" {
		r.URL.Scheme = "http"
	}
	if r.URL.Host == "" {
		r.URL.Host = "example.com"
	}

	// Use `DumpRequestOut` instead of `DumpRequest` to preserve the
	// querystring.
	req, err := httputil.DumpRequestOut(r, true)
	if err != nil {
		return nil, err
	}
	req = NormalizeNewlines(req)
	req = bytes.TrimSpace(req)

	return req, nil
}

func DumpResponse(w *http.Response) ([]byte, error) {
	b, err := io.ReadAll(w.Body)
	if err != nil {
		return nil, err
	}

	b, err = prettyBytes(b)
	if err != nil {
		return nil, err
	}

	w.Body = io.NopCloser(bytes.NewReader(b))

	res, err := httputil.DumpResponse(w, true)
	if err != nil {
		return nil, err
	}

	res = NormalizeNewlines(res)
	res = bytes.TrimSpace(res)

	return res, nil
}

// NormalizeNewlines normalizes \r\n (windows) and \r (mac)
// into \n (unix)
// Reference [here].
// [here]: https://www.programming-books.io/essential/go/normalize-newlines-1d3abcf6f17c4186bb9617fa14074e48
func NormalizeNewlines(d []byte) []byte {
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = bytes.Replace(d, []byte{13, 10}, []byte{10}, -1)
	// replace CF \r (mac) with LF \n (unix)
	d = bytes.Replace(d, []byte{13}, []byte{10}, -1)
	return d
}

type Dump struct {
	Line    string
	Headers http.Header
	Body    []byte
}

func Parse(req []byte) (*Dump, error) {
	req = bytes.TrimSpace(req)
	rawReqLine, rawHeadersAndBody, _ := bytes.Cut(req, LineBreak)
	rawHeaders, body, _ := bytes.Cut(rawHeadersAndBody, bytes.Repeat(LineBreak, 2))
	headers, err := parseHeaders(rawHeaders)
	if err != nil {
		return nil, err
	}

	return &Dump{
		Line:    string(bytes.TrimSpace(rawReqLine)),
		Headers: headers,
		Body:    bytes.TrimSpace(body),
	}, nil
}

// parseHeaders parse the HTTP headers from key-value strings into map of
// strings.
func parseHeaders(headers []byte) (http.Header, error) {
	headers = bytes.TrimSpace(headers)

	h := make(http.Header)
	kvs := bytes.Split(headers, LineBreak)
	for _, kv := range kvs {
		k, v, ok := bytes.Cut(kv, SemiColon)
		if !ok {
			return nil, fmt.Errorf("invalid header format: %q", kv)
		}
		ks := string(bytes.TrimSpace(k))
		vs := string(bytes.TrimSpace(v))
		h[ks] = append(h[ks], vs)
	}

	return h, nil
}

func prettyBytes(b []byte) ([]byte, error) {
	if !json.Valid(b) {
		return b, nil
	}

	bb := new(bytes.Buffer)
	if err := json.Indent(bb, b, "", " "); err != nil {
		return nil, err
	}

	return bb.Bytes(), nil
}
