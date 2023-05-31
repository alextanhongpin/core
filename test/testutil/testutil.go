package testutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-cmp/cmp"
)

// JSONDump captures the HTTP response and saves it into a file for
// comparison.
func JSONDump(r *http.Request, handler http.HandlerFunc, out string, statusCode int, opts ...cmp.Option) error {
	w := httptest.NewRecorder()

	handler(w, r)
	res := w.Result()
	defer res.Body.Close()

	if want, got := statusCode, res.StatusCode; want != got {
		return fmt.Errorf("want status code %d, got %d", want, got)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return Dump(b, out, opts...)
}

// Dump saves the value as a json file.
func Dump(v any, name string, opts ...cmp.Option) error {
	got, err := marshal(v)
	if err != nil {
		return err
	}

	if err := writeIfNotExists(got, name); err != nil {
		return err
	}

	want, err := os.ReadFile(name)
	if err != nil {
		return err
	}

	return CmpJSON(want, got, opts...)
}

func CmpJSON(a, b []byte, opts ...cmp.Option) error {
	var want, got map[string]any
	if err := json.Unmarshal(a, &want); err != nil {
		return err
	}

	if err := json.Unmarshal(b, &got); err != nil {
		return err
	}

	// NOTE: The want and got is reversed here.
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		return fmt.Errorf("want(+), got(-): %s", diff)
	}

	return nil
}

func IsJSONTime(v any) error {
	ts, ok := v.(string)
	if !ok {
		return fmt.Errorf("want time string, got %v", v)
	}

	_, err := time.Parse(time.RFC3339, ts)
	return err
}

func writeIfNotExists(body []byte, name string) error {
	f, err := os.OpenFile(name, os.O_RDONLY, 0644)
	if errors.Is(err, os.ErrNotExist) {
		dir := filepath.Dir(name)

		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		} // Create your file

		return os.WriteFile(name, body, 0644)
	}
	if err != nil {
		return err
	}

	defer f.Close()

	return nil
}

func marshal(v any) ([]byte, error) {
	b, ok := v.([]byte)
	if ok {
		var bb bytes.Buffer
		if err := json.Indent(&bb, b, "", " "); err != nil {
			return nil, err
		}

		return bb.Bytes(), nil
	}

	return json.MarshalIndent(v, "", " ")
}
