package maputil_test

import (
	"sort"
	"testing"

	"github.com/alextanhongpin/core/test/testutil"
	"github.com/alextanhongpin/core/types/maputil"
	"github.com/stretchr/testify/assert"
)

func TestFieldsReplacer(t *testing.T) {
	type Name string

	type Hobby struct {
		Name string `json:"name"`
	}

	type User struct {
		Name    Name            `json:"name"`
		Age     int             `json:"age"`
		Hobbies []Hobby         `json:"hobbies"`
		Meta    map[string]Name `json:"meta"`
		Tags    []string        `json:"tags"`
	}

	u := User{
		Name:    "john",
		Age:     10,
		Hobbies: []Hobby{{Name: "dancing"}, {Name: "coding"}},
		Meta: map[string]Name{
			"foo": Name("bar"),
		},
		Tags: []string{"user", "name"},
	}

	m, err := maputil.ToMap(u)
	if err != nil {
		panic(err)
	}

	// Create an obfuscator.
	var fields []string
	mr := maputil.ReplaceStringFunc(m, func(k, v string) string {
		fields = append(fields, k)
		if k == "name" {
			return "EDITED:" + v
		}

		if k == "hobbies[_].name" {
			return "EDITED:" + v
		}

		return v
	})

	assert := assert.New(t)
	want := []string{
		"name",
		"hobbies[_].name",
		"hobbies[_].name",
		"meta.foo",
		"tags[_]",
		"tags[_]",
	}
	sort.Strings(want)
	sort.Strings(fields)
	assert.Equal(want, fields)
	testutil.DumpJSON(t, m, testutil.FileName("before"))
	testutil.DumpJSON(t, mr, testutil.FileName("after"))
}
