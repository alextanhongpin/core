package yaml_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	codec "github.com/alextanhongpin/core/codec/yaml"
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
		{"parent within root", map[string]string{"main.yaml": "!include sub/a.yaml", "sub/a.yaml": "!include ../b.yaml", "b.yaml": "b: 1"}, map[string]any{"b": 1}, ""},
		{"cycle through dot segments", map[string]string{"main.yaml": "!include sub/../main.yaml"}, nil, "cycle"},
		{"parent escape", map[string]string{"main.yaml": "!include ../secret.yaml"}, nil, "escapes"},
		{"merge escape", map[string]string{"main.yaml": "\"!merge\": sub/../../secret.yaml"}, nil, "escapes"},
		{"absolute path", map[string]string{"main.yaml": "!include /etc/hostname"}, nil, "escapes"},
		{"fan-out limit", map[string]string{
			"main.yaml": "!include [a.yaml, a.yaml, a.yaml, a.yaml, a.yaml, a.yaml, a.yaml, a.yaml, a.yaml, a.yaml]",
			"a.yaml":    "!include [b.yaml, b.yaml, b.yaml, b.yaml, b.yaml, b.yaml, b.yaml, b.yaml, b.yaml, b.yaml]",
			"b.yaml":    "!include [c.yaml, c.yaml, c.yaml, c.yaml, c.yaml, c.yaml, c.yaml, c.yaml, c.yaml, c.yaml]",
			"c.yaml":    "c: 1",
		}, nil, "more than 1000 files"},
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

func TestLoadSymlinks(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.yaml"), []byte("secret: 1"), 0600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	write := func(name, data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	link := func(target, name string) {
		t.Helper()
		if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
			t.Skip("symlinks unsupported:", err)
		}
	}

	t.Run("escape", func(t *testing.T) {
		link(filepath.Join(outside, "secret.yaml"), "secret.yaml")
		write("escape.yaml", "!include secret.yaml")
		var got map[string]any
		if err := codec.LoadExtendedYAML(filepath.Join(dir, "escape.yaml"), &got); err == nil {
			t.Fatalf("symlink escape was followed: %v", got)
		}
	})

	t.Run("directory loop", func(t *testing.T) {
		link(".", "loop")
		write("loop.yaml", "!include loop/loop.yaml")
		var got map[string]any
		err := codec.LoadExtendedYAML(filepath.Join(dir, "loop.yaml"), &got)
		// Stopped by the depth limit or the OS symlink limit, whichever is first.
		if err == nil {
			t.Fatalf("symlink loop terminated without error: %v", got)
		}
	})
}

func TestLoadFS(t *testing.T) {
	fsys := fstest.MapFS{
		"app/main.yaml":    {Data: []byte("\"!merge\": ../shared/base.yaml\nname: app")},
		"shared/base.yaml": {Data: []byte("name: base\nport: 80")},
	}
	var got map[string]any
	if err := codec.Load(fsys, "app/main.yaml", &got); err != nil {
		t.Fatal(err)
	}
	if want := map[string]any{"name": "app", "port": 80}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}

	err := codec.Load(fsys, "app/missing.yaml", &got)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("want fs.ErrNotExist, got %v", err)
	}
	if err := codec.Load(fsys, "../main.yaml", &got); !errors.Is(err, codec.ErrPathEscape) {
		t.Fatalf("want ErrPathEscape, got %v", err)
	}
}

func TestLoadSample(t *testing.T) {
	var got struct {
		Name   string            `yaml:"name"`
		Config map[string]string `yaml:"config"`
	}
	if err := codec.LoadExtendedYAML("testdata/main.yaml", &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "bar" || !reflect.DeepEqual(got.Config, map[string]string{"foo": "bar", "bar": "baz"}) {
		t.Fatalf("unexpected sample configuration: %#v", got)
	}
}
