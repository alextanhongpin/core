package maputil_test

import (
	"sort"
	"testing"

	"github.com/alextanhongpin/core/test/testutil"
	"github.com/alextanhongpin/core/types/maputil"
	"github.com/stretchr/testify/assert"
)

func TestReplace(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		m := testReplaceFixture(t)

		var keys []string
		got := maputil.ReplaceFunc(m, func(k string, v bool) bool {
			keys = append(keys, k)
			return !v
		})

		want := []string{"isMarried", "meta.isActive"}
		sort.Strings(want)
		sort.Strings(keys)

		assert.Equal(t, want, keys)
		testutil.DumpJSON(t, m, testutil.FileName("before"))
		testutil.DumpJSON(t, got, testutil.FileName("after"))
	})

	t.Run("string", func(t *testing.T) {
		m := testReplaceFixture(t)

		var keys []string
		got := maputil.ReplaceFunc(m, func(k, v string) string {
			keys = append(keys, k)
			return "EDITED: " + v
		})

		want := []string{
			"name",
			"hobbies[_].name",
			"hobbies[_].name",
			"meta.name",
			"tags[_]",
			"tags[_]",
		}
		sort.Strings(want)
		sort.Strings(keys)

		assert.Equal(t, want, keys)
		testutil.DumpJSON(t, m, testutil.FileName("before"))
		testutil.DumpJSON(t, got, testutil.FileName("after"))
	})

	t.Run("float64", func(t *testing.T) {
		m := testReplaceFixture(t)

		var keys []string
		got := maputil.ReplaceFunc(m, func(k string, v float64) float64 {
			keys = append(keys, k)
			return v + 10.0

		})
		want := []string{
			"age",
			"height",
			"meta.age",
			"meta.ids[_]",
			"meta.ids[_]",
			"meta.ids[_]",
		}

		sort.Strings(want)
		sort.Strings(keys)
		assert.Equal(t, want, keys)
		testutil.DumpJSON(t, m, testutil.FileName("before"))
		testutil.DumpJSON(t, got, testutil.FileName("after"))
	})
}

func testReplaceFixture(t *testing.T) map[string]any {
	t.Helper()

	type Name string

	type Hobby struct {
		Name string `json:"name"`
	}

	type User struct {
		Name      Name           `json:"name"`
		Age       int            `json:"age"`
		Hobbies   []Hobby        `json:"hobbies"`
		IsMarried bool           `json:"isMarried"`
		Height    int64          `json:"height"`
		Meta      map[string]any `json:"meta"`
		Tags      []string       `json:"tags"`
	}

	u := User{
		Name: "john",
		Age:  10,
		Hobbies: []Hobby{
			{Name: "dancing"},
			{Name: "coding"},
		},
		IsMarried: true,
		Height:    168,
		Meta: map[string]any{
			"name":     Name("bar"),
			"age":      10,
			"isActive": true,
			"ids":      []float64{1.0, 2.0, 3.0},
		},
		Tags: []string{"user", "name"},
	}

	m, err := maputil.StructToMap(u)
	if err != nil {
		t.Fatal(err)
	}

	return m
}
