package yaml_test

import (
	codec "github.com/core/codec/yaml"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  map[string]any
		err   string
	}{
		{"nested", map[string]string{"main.yaml": "config: !include sub/config.yaml", "sub/config.yaml": "db: !include db.yaml", "sub/db.yaml": "host: localhost"}, map[string]any{"config": map[string]any{"db": map[string]any{"host": "localhost"}}}, ""},
		{"root multi-include", map[string]string{"main.yaml": "!include [a.yaml, b.yaml, c.yaml]", "a.yaml": "a: 1", "b.yaml": "b: 2", "c.yaml": "b: 3"}, map[string]any{"a": 1, "b": 3}, ""},
		{"ordered merge", map[string]string{"main.yaml": "before: local\n\"!merge\": base.yaml\nafter: local", "base.yaml": "before: base\nafter: base"}, map[string]any{"before": "base", "after": "local"}, ""},
		{"sequence", map[string]string{"main.yaml": "items: !include [a.yaml, b.yaml]", "a.yaml": "[a, b]", "b.yaml": "[c]"}, map[string]any{"items": []any{"a", "b", "c"}}, ""},
		{"aliases", map[string]string{"main.yaml": "a: &a !include a.yaml\nb: *a", "a.yaml": "value: 1"}, map[string]any{"a": map[string]any{"value": 1}, "b": map[string]any{"value": 1}}, ""},
		{"cycle", map[string]string{"main.yaml": "!include a.yaml", "a.yaml": "!include main.yaml"}, nil, "cycle"},
		{"empty list", map[string]string{"main.yaml": "a: !include []"}, nil, "at least one"},
		{"invalid filename", map[string]string{"main.yaml": "a: !include [12]"}, nil, "filename strings"},
		{"empty file", map[string]string{"main.yaml": ""}, nil, "EOF"},
		{"multiple documents", map[string]string{"main.yaml": "a: 1\n---\nb: 2"}, nil, "multiple YAML documents"},
		{"duplicate keys", map[string]string{"main.yaml": "a: 1\na: 2"}, nil, "duplicate key"},
		{"mismatched kinds", map[string]string{"main.yaml": "!include [a.yaml, b.yaml]", "a.yaml": "[1]", "b.yaml": "a: 1"}, nil, "matching"},
		{"invalid merge", map[string]string{"main.yaml": "\"!merge\": a.yaml", "a.yaml": "[1]"}, nil, "must contain a mapping"},
		{"missing file", map[string]string{"main.yaml": "!include missing.yaml"}, nil, "missing.yaml"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, data := range tc.files {
				path := filepath.Join(dir, name)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var got map[string]any
			err := codec.LoadExtendedYAML(filepath.Join(dir, "main.yaml"), &got)
			if tc.err != "" {
				if err == nil || !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("want %q, got %v", tc.err, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}
