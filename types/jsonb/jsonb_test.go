package jsonb_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/types/jsonb"
)

func TestJSONB_ReviverGet(t *testing.T) {
	m := map[string]any{
		"key1": map[string]any{
			"key2": map[string]any{
				"strings": []any{"foo", "bar", "baz"},
				"maps": []any{
					map[string]any{
						"key3": "val3",
					},
				},
			},
			"key4": "value4",
		},
	}
	err := jsonb.Reviver(m, func(path jsonb.Path, value any) (any, error) {
		fmt.Println(path.String(), value)
		return value, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestJSONB_ReviverSet(t *testing.T) {
	m := map[string]any{
		"key1": map[string]any{
			"key2": map[string]any{
				"strings": []any{"foo", "bar", "baz"},
				"maps": []any{
					map[string]any{
						"key3": "val3",
					},
				},
			},
			"key4": "value4",
		},
	}
	err := jsonb.Reviver(m, func(path jsonb.Path, value any) (any, error) {
		if path.Match("key1.key2.maps[].key3") {
			return "value3", nil
		}
		return value, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(m)
}

func TestJSONB_Get(t *testing.T) {
	m := map[string]any{
		"key1": map[string]any{
			"key2": map[string]any{
				"strings": []any{"foo", "bar", "baz"},
				"maps": []any{
					map[string]any{
						"key3": "val3",
					},
				},
			},
			"key4": "value4",
		},
	}
	res, err := jsonb.Get(m, "key1.key2")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(res)
}

func TestJSONB_Set(t *testing.T) {
	m := map[string]any{
		"key1": map[string]any{
			"key2": map[string]any{
				"strings": []any{"foo", "bar", "baz"},
				"maps": []any{
					map[string]any{
						"key3": "val3",
					},
				},
			},
			"key4": "value4",
		},
	}
	err := jsonb.Set(m, "key1.key2.strings", []any{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(m)

	err = jsonb.Set(m, "key1.key2.strings", "string")
	if !errors.Is(err, jsonb.ErrType) {
		t.Fatal(err)
	}
	t.Log(m)
}
