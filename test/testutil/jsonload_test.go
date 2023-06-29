package testutil_test

import (
	"testing"

	"github.com/alextanhongpin/core/test/testutil"
	"github.com/stretchr/testify/assert"
)

func TestLoadJSON(t *testing.T) {
	type Person struct {
		Name      string `json:"name"`
		Age       int64  `json:"age"`
		IsMarried bool   `json:"isMarried"`
	}

	t.Run("load json", func(t *testing.T) {
		assert := assert.New(t)
		p, err := testutil.LoadJSONFile[Person]("./testdata/TestDumpJSON/person.json")
		assert.Nil(err)
		assert.Equal("John Appleseed", p.Name)
		assert.Equal(int64(13), p.Age)
		assert.True(p.IsMarried)
	})

	t.Run("disallow unknown fields option", func(t *testing.T) {
		assert := assert.New(t)
		p, err := testutil.LoadJSONFile[Person]("./testdata/TestDumpJSON/person.json", testutil.DisallowUnknownFields())
		assert.NotNil(err)
		assert.Equal(`json: unknown field "bornAt"`, err.Error())
		assert.Nil(p)
	})
}
