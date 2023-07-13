package httpdump

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
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
	// Just don't forgot to strip of the user-agent=Go-http-client/1.1 and accept-encoding=gzip
	b, err := httputil.DumpRequest(r, true)
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
	// httputil.DumpRequest seem to strip the querystring
	// when constructed with httptest.NewRequest.
	if len(r.URL.RequestURI()) > len(r.RequestURI) {
		r.RequestURI = r.URL.RequestURI()
	}

	if r.Body == nil {
		return r, nil
	}

	// Prettify the request body.
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	b, err = bytesPretty(b)
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewReader(b))

	n := strconv.Itoa(len(b))
	if o := r.Header.Get("Content-Length"); o != n && len(b) > 0 {
		// Update the content length.
		r.Header.Set("Content-Length", n)
	}

	return r, nil
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
