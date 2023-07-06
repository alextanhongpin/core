package httpdump

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
)

func ReadRequest(b []byte) (*http.Request, error) {
	b = bytes.TrimSpace(b)
	b = denormalizeNewlines(b)

	r, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(b)))
	if err != nil {
		return nil, err
	}

	return normalizeRequest(r)
}

func DumpRequest(r *http.Request) ([]byte, error) {
	r, err := normalizeRequest(r)
	if err != nil {
		return nil, err
	}

	// Use `DumpRequestOut` instead of `DumpRequest` to preserve the querystring.
	b, err := httputil.DumpRequestOut(r, true)
	if err != nil {
		return nil, err
	}

	b = normalizeNewlines(b)

	return b, nil
}

func FromRequest(r *http.Request) (*Dump, error) {
	return requestToDump(r)
}

func normalizeRequest(r *http.Request) (*http.Request, error) {
	req := r.Clone(r.Context())

	// Prettify the request body.
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	b, err = bytesPretty(b)
	if err != nil {
		return nil, err
	}

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

func requestToDump(r *http.Request) (*Dump, error) {
	reqLine := formatRequestLine(r)

	b, err := io.ReadAll(r.Body)
	if err != nil {
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

	r.Body = io.NopCloser(bytes.NewReader(b))

	return &Dump{
		Line:   reqLine,
		Header: r.Header.Clone(),
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

func valueOrDefault(v, d string) string {
	if v != "" {
		return v
	}

	return d
}
