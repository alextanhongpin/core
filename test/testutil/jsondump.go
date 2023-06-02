package testutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// DumpJSON dumps a type as json.
func DumpJSON(t *testing.T, v any, opts ...Option) {
	t.Helper()

	got, err := marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	fileName := fmt.Sprintf("./testdata/%s.json", t.Name())
	if err := writeToNewFile(fileName, got); err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatal(err)
	}
	o := new(Options)
	for _, opt := range opts {
		opt(o)
	}

	if err := cmpJSON(want, got, o.bodyopts...); err != nil {
		t.Fatal(err)
	}
}

func cmpJSON(a, b []byte, opts ...cmp.Option) error {
	var want, got map[string]any
	if err := json.Unmarshal(a, &want); err != nil {
		return err
	}

	if err := json.Unmarshal(b, &got); err != nil {
		return err
	}

	// NOTE: The want and got is reversed here.
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		return diffError(diff)
	}

	return nil
}

func writeToNewFile(name string, body []byte) error {
	f, err := os.OpenFile(name, os.O_RDONLY, 0644)
	if errors.Is(err, os.ErrNotExist) {
		dir := filepath.Dir(name)

		if err := os.MkdirAll(dir, 0700); err != nil && !os.IsExist(err) {
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
		if !json.Valid(b) {
			return b, nil
		}

		var bb bytes.Buffer
		if err := json.Indent(&bb, b, "", " "); err != nil {
			return nil, err
		}

		return bb.Bytes(), nil
	}

	return json.MarshalIndent(v, "", " ")
}

func diffError(diff string) error {
	if diff == "" {
		return nil
	}

	return fmt.Errorf("want(+), got(-):\n\n%s", diff)
}
