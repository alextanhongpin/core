package jsonb_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/types/jsonb"
)

func TestJSONB(t *testing.T) {
	b := []byte(`{"foo": "bar", "tags": [{"foo": "bar"}, {"foo": "baz"}], "a": "b"}`)
	var a any
	err := json.Unmarshal(b, &a)
	if err != nil {
		t.Fatal(err)
	}
	err = jsonb.Reviver(a, func(path jsonb.Path, value any) error {
		fmt.Println(path.String(), value)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
