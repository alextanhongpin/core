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
	"strings"
)

func ReadResponse(b []byte) (*http.Response, error) {
	b = bytes.TrimSpace(b)
	b = denormalizeNewlines(b)

	r, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(b)), nil)
	if err != nil {
		return nil, err
	}

	return normalizeResponse(r)
}

func DumpResponse(r *http.Response) ([]byte, error) {
	r, err := normalizeResponse(r)
	if err != nil {
		return nil, err
	}

	b, err := httputil.DumpResponse(r, true)
	if err != nil {
		return nil, err
	}

	b = normalizeNewlines(b)
	return b, nil
}

func FromResponse(r *http.Response) (*Dump, error) {
	return responseToDump(r)
}

func normalizeResponse(res *http.Response) (*http.Response, error) {
	// Prettify the request body.
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	b, err = bytesPretty(b)
	if err != nil {
		return nil, err
	}

	res.Body = io.NopCloser(bytes.NewReader(b))

	// If the content-length is set, we need to update it.
	n := int64(len(b))
	if res.ContentLength > 0 && res.ContentLength != n {
		res.ContentLength = n
	}

	return res, nil
}

func responseToDump(r *http.Response) (*Dump, error) {
	resLine := formatResponseLine(r)

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
		Line:    resLine,
		Header:  r.Header.Clone(),
		Body:    a,
		Trailer: r.Trailer.Clone(),
	}, nil
}

func formatResponseLine(r *http.Response) string {
	// Status line
	text := r.Status
	if text == "" {
		text = http.StatusText(r.StatusCode)
		if text == "" {
			text = "status code " + strconv.Itoa(r.StatusCode)
		}
	} else {
		// Just to reduce stutter, if user set r.Status to "200 OK" and StatusCode to 200.
		// Not important.
		text = strings.TrimPrefix(text, strconv.Itoa(r.StatusCode)+" ")
	}

	return fmt.Sprintf("HTTP/%d.%d %03d %s", r.ProtoMajor, r.ProtoMinor, r.StatusCode, text)
}
