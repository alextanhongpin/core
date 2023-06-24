package maputil_test

import (
	"sort"
	"testing"

	"github.com/alextanhongpin/core/types/maputil"
	"github.com/stretchr/testify/assert"
)

func TestKeys(t *testing.T) {
	m := map[string]any{
		"foo": "bar",
		"1": map[string]any{
			"2": "two",
			"3": map[string]any{
				"z": "end",
			},
		},
		"tags": []string{
			"1",
			"2",
			"3",
		},
		"nested": []any{
			map[string]any{"a": "b"},
			map[string]any{"c": "d"},
		},
	}

	t.Run("keys", func(t *testing.T) {
		keys := maputil.Keys(m)
		want := []string{"foo", "1", "tags", "nested"}

		sort.Strings(keys)
		sort.Strings(want)
		assert.Equal(t, want, keys)
	})

	t.Run("all keys", func(t *testing.T) {
		keys := maputil.AllKeys(m)
		want := []string{
			"foo",
			"1",
			"1.2",
			"1.3",
			"1.3.z",
			"tags",
			"nested",
			"nested[_]",
			"nested[_].a",
			"nested[_].c",
		}

		sort.Strings(keys)
		sort.Strings(want)
		assert.Equal(t, want, keys)
	})
}
