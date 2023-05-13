package snapshottest

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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func CmpJSON(t *testing.T, a, b []byte, opts ...cmp.Option) {
	var l, r map[string]any
	if err := json.Unmarshal(a, &l); err != nil {
		t.Error(err)
	}

	if err := json.Unmarshal(b, &r); err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(l, r, opts...); diff != "" {
		t.Errorf("want(+), got(-): %s", diff)
	}
}

func HTTPSnapshot(t *testing.T, r *http.Request, handler http.HandlerFunc, out string, statusCode int, opts ...cmp.Option) {
	w := httptest.NewRecorder()

	handler(w, r)
	res := w.Result()
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}

	Snapshot(t, b, out, opts...)
}

func Snapshot(t *testing.T, v any, name string, opts ...cmp.Option) {
	if err := snapshot(v, name); err != nil {
		t.Error(err)
	}

	got, err := marshal(v)
	if err != nil {
		t.Error(err)
	}

	want, err := os.ReadFile(name)
	if err != nil {
		t.Error(err)
	}

	CmpJSON(t, want, got, opts...)
}

func snapshot(v any, name string) error {
	b, err := marshal(v)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(name, os.O_RDONLY, 0644)
	if errors.Is(err, os.ErrNotExist) {
		dir := filepath.Dir(name)

		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		} // Create your file

		return os.WriteFile(name, b, 0644)
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
		br := bytes.NewBuffer(nil)
		if err := json.Indent(br, b, "", " "); err != nil {
			return nil, err
		}

		return br.Bytes(), nil
	}

	return json.MarshalIndent(v, "", " ")
}

func IsJsonTime(t *testing.T, v any) bool {
	ts, ok := v.(string)
	if !ok {
		t.Errorf("want time string, got %v", v)

		return false
	}

	_, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		fmt.Printf("JSON %q, %q %v\n", v, ts, err)
		t.Error(err)

		return false
	}

	return true
}
