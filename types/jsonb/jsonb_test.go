package jsonb_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/alextanhongpin/core/types/jsonb"
)

func decode(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func encode(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSet(t *testing.T) {
	tests := []struct {
		name, data, key, want string
	}{
		{"dict", `{"foo":{"bar":{"baz":"value"}}}`, "foo.bar.baz", `{"foo":{"bar":{"baz":"edited"}}}`},
		{"array", `[{"foo":{"baz":"value"}}]`, "[].foo.baz", `[{"foo":{"baz":"edited"}}]`},
		{"index", `{"a":["x","y"]}`, "a[]", `{"a":["edited","edited"]}`},
		{"wildcard", `{"a":{"k":"x"},"b":{"k":"y"}}`, "*.k", `{"a":{"k":"edited"},"b":{"k":"edited"}}`},
		{"missing key is not added", `{"foo":{}}`, "foo.bar", `{"foo":{}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := decode(t, tt.data)
			if err := jsonb.Set(data, tt.key, "edited"); err != nil {
				t.Fatal(err)
			}
			if got := encode(t, data); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSetTypeMismatch(t *testing.T) {
	data := decode(t, `{"n":1}`)
	err := jsonb.Set(data, "n", 2) // decoded JSON numbers are float64
	if !errors.Is(err, jsonb.ErrType) {
		t.Fatalf("got %v, want ErrType", err)
	}
	if err := jsonb.Set(data, "n", 2.0); err != nil {
		t.Fatal(err)
	}
	if got := encode(t, data); got != `{"n":2}` {
		t.Fatalf("got %s", got)
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name, data, key, want string
	}{
		{"dict key", `{"foo":{"bar":{"baz":"value"}}}`, "foo.bar.baz", `{"foo":{"bar":{}}}`},
		{"array dict key", `[{"foo":{"bar":{"baz":"value"}}}]`, "[].foo.bar.baz", `[{"foo":{"bar":{}}}]`},
		{"array elements", `{"a":[1,2,3]}`, "a[]", `{"a":[]}`},
		{"top-level key", `{"a":1,"b":2}`, "a", `{"b":2}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := decode(t, tt.data)
			if err := jsonb.Delete(data, tt.key); err != nil {
				t.Fatal(err)
			}
			if got := encode(t, data); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDeleteSelectedArrayElement(t *testing.T) {
	data := decode(t, `{"a":[{"id":1},{"id":2},{"id":3}]}`)
	err := jsonb.Reviver(data, func(path jsonb.Path, v any) (any, error) {
		if m, ok := v.(map[string]any); ok && path.Match("a[]") && m["id"] == 2.0 {
			return nil, jsonb.ErrDelete
		}
		return v, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := encode(t, data); got != `{"a":[{"id":1},{"id":3}]}` {
		t.Fatalf("got %s", got)
	}
}

func TestDeleteRootArrayElement(t *testing.T) {
	data := decode(t, `[1,2]`)
	if err := jsonb.Delete(data, "[]"); !errors.Is(err, jsonb.ErrDelete) {
		t.Fatalf("got %v, want ErrDelete", err)
	}
	if got := encode(t, data); got != `[1,2]` {
		t.Fatalf("root array changed: %s", got)
	}
}

// Regression: traversal used to delete every map entry holding JSON null.
func TestNullValuesArePreserved(t *testing.T) {
	data := decode(t, `{"a":null,"b":{"c":null}}`)
	if _, err := jsonb.Get(data, "b"); err != nil {
		t.Fatal(err)
	}
	if got := encode(t, data); got != `{"a":null,"b":{"c":null}}` {
		t.Fatalf("got %s", got)
	}
}

// Regression: array elements used to be visited twice, as "[]" and "[i]",
// so a "*" pattern returned each element twice.
func TestGetVisitsArrayElementsOnce(t *testing.T) {
	data := decode(t, `[1,2]`)
	got, err := jsonb.Get(data, "*")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got) != "[1 2]" {
		t.Fatalf("got %v", got)
	}
}

func TestReviverPaths(t *testing.T) {
	data := decode(t, `{"a":[{"b":1}],"c":[[2]]}`)
	var paths []string
	err := jsonb.Reviver(data, func(path jsonb.Path, v any) (any, error) {
		paths = append(paths, path.String())
		return v, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	want := `[ a a[0] a[0].b c c[0] c[0][0]]`
	if fmt.Sprint(paths) != want {
		t.Fatalf("got %v, want %s", paths, want)
	}
}

func TestReviverSkip(t *testing.T) {
	data := decode(t, `[1,2,3]`)
	var seen []any
	err := jsonb.Reviver(data, func(path jsonb.Path, v any) (any, error) {
		seen = append(seen, v)
		if v == 2.0 {
			return v, jsonb.ErrSkip
		}
		return v, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(seen) != "[1 2]" {
		t.Fatalf("got %v", seen)
	}
}

func TestPathMatch(t *testing.T) {
	tests := []struct {
		path    jsonb.Path
		pattern string
		want    bool
	}{
		{jsonb.Path{"a", "b"}, "a.b", true},
		{jsonb.Path{"a", "b"}, "a.*", true},
		{jsonb.Path{"a", "b"}, "a", false},
		{jsonb.Path{"items[12]"}, "items[]", true},
		{jsonb.Path{"items[12]"}, "items[12]", true},
		{jsonb.Path{"items[12]"}, "items[1]", false},
		{jsonb.Path{"items"}, "items[]", false},
		{jsonb.Path{"m[0][3]"}, "m[][]", true},
		{jsonb.Path{"m[x]"}, "m[]", false},
		{jsonb.Path{"[0]"}, "[]", true},
	}
	for _, tt := range tests {
		if got := tt.path.Match(tt.pattern); got != tt.want {
			t.Errorf("%v.Match(%q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
		}
	}
}

func ExampleGet() {
	var data any
	_ = json.Unmarshal([]byte(`{"users":[{"name":"Alice"},{"name":"Bob"}]}`), &data)

	names, _ := jsonb.Get(data, "users[].name")
	fmt.Println(names)

	_ = jsonb.Set(data, "users[].name", "redacted")
	b, _ := json.Marshal(data)
	fmt.Println(string(b))
	// Output:
	// [Alice Bob]
	// {"users":[{"name":"redacted"},{"name":"redacted"}]}
}
