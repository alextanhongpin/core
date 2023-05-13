package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// DotHTTPDump captures the HTTP response and saves it into a file in .http
// format for comparison.
func DotHTTPDump(r *http.Request, handler http.HandlerFunc, out string, statusCode int, opts ...cmp.Option) error {
	w := httptest.NewRecorder()

	handler(w, r)
	ws := w.Result()
	defer ws.Body.Close()

	// We need to assign the header.
	ws.Header = w.Header()

	if want, got := statusCode, ws.StatusCode; want != got {
		return fmt.Errorf("want status code %d, got %d", want, got)
	}

	got, err := io.ReadAll(ws.Body)
	if err != nil {
		return err
	}
	ws.Body = io.NopCloser(bytes.NewReader(got))

	newDotHTTP := format(ws, r)
	if err := writeIfNotExists([]byte(newDotHTTP), out); err != nil {
		return err
	}

	dotHTTP, err := os.ReadFile(out)
	if err != nil {
		return err
	}

	want, err := parseResponse(ws, string(dotHTTP))
	if err != nil {
		return err
	}

	if want == "" {
		return nil
	}

	return CmpJSON([]byte(want), got, opts...)
}

func format(w *http.Response, r *http.Request) string {
	var output []string

	output = append(output, formatRequest(r)...)
	output = append(output, "")
	output = append(output, formatResponse(w)...)

	return strings.Join(output, "\n")
}

func formatRequest(r *http.Request) []string {
	var output []string

	// E.g. GET http://example.com
	output = append(output, fmt.Sprintf("%s %s", r.Method, formatURL(r)))

	// E.g.
	// ?name=john
	// &age10
	// &is_married=true
	if q := r.URL.RawQuery; len(q) > 0 {
		output = append(output, formatQuery(q))
	}

	// Append headers.
	if len(r.Header) > 0 {
		output = append(output, formatHeader(r.Header))
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	if len(b) != 0 && string(b) != "<nil>" {
		// Newline.
		output = append(output, "")

		// Get Request body, which can be either
		// - form
		// - json
		// By default, we assume it as json
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			panic(err)
		}
		_ = params
		switch mediaType {
		case "application/x-www-form-urlencoded":
			output = append(output, formatQuery(string(b)))
		default:
			var bb bytes.Buffer
			if err := json.Indent(&bb, b, "", " "); err != nil {
				panic(fmt.Errorf("%w: failed to indent %q", err, b))
			}
			output = append(output, bb.String())
		}
	}
	return output
}

func formatResponse(w *http.Response) []string {
	var output []string

	// E.g. HTTP/1.1 200 - OK
	output = append(output, fmt.Sprintf("%s %d - %s", w.Proto, w.StatusCode, http.StatusText(w.StatusCode)))
	if len(w.Header) > 0 {
		output = append(output, formatHeader(w.Header))
	}

	// Read response body.
	buf := &bytes.Buffer{}
	tee := io.TeeReader(w.Body, buf)
	defer w.Body.Close()

	b, err := io.ReadAll(tee)
	if err != nil {
		panic(err)
	}

	w.Body = io.NopCloser(buf)
	if len(b) != 0 && string(b) != "<nil>" {
		contentType := w.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/json"
		}
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			panic(err)
		}
		_ = params

		switch mediaType {
		case "application/json":
			// Newline.
			output = append(output, "")
			var bb bytes.Buffer
			if err := json.Indent(&bb, b, "", " "); err != nil {
				panic(fmt.Errorf("%w: failed to indent %q", err, b))
			}
			output = append(output, bb.String())
		default:
			// Assume it is text.
			// Newline.
			output = append(output, "")
			output = append(output, string(b))
		}
	}

	return output
}

func formatHeader(headers map[string][]string) string {
	// Sort
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}

	var res []string
	for _, key := range keys {
		for _, val := range headers[key] {
			res = append(res, fmt.Sprintf("%s: %s", key, val))
		}
	}

	return strings.Join(res, "\n")
}

// formatURL formats the URL without the query
// http://www.example.com/user
func formatURL(r *http.Request) string {
	scheme := "http"
	if r.URL.Scheme != "" {
		scheme = r.URL.Scheme
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)
}

/*
	formatQuery

id=1&name=john&age=10
?id=1
&name=john
&age=10
*/
func formatQuery(rawQuery string) string {
	parts := strings.Split(rawQuery, "&")
	parts[0] = "?" + parts[0]
	return strings.Join(parts, "\n&")
}

// parseResponse attempts to extract the json body from
// the dotHTTP file format.
func parseResponse(w *http.Response, response string) (string, error) {
	contentType := w.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		panic(err)
	}
	_ = params

	if mediaType != "application/json" {
		return "", nil
	}

	var output []string

	// E.g. HTTP/1.1 200 - OK
	output = append(output, fmt.Sprintf("%s %d - %s", w.Proto, w.StatusCode, http.StatusText(w.StatusCode)))
	if len(w.Header) > 0 {
		output = append(output, formatHeader(w.Header))
	}

	responseWithoutBody := strings.Join(output, "\n")
	if _, after, ok := strings.Cut(response, responseWithoutBody); ok {
		return strings.TrimSpace(after), nil
	}

	return "", nil
}
