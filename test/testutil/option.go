package testutil

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const (
	FormatYAML Format = "yaml"
	FormatJSON Format = "json"
)

// Format represents the embedded format.
type Format string

func (Format) isSQL() {}

type HTTPOption interface {
	isHTTP()
}

type JSONOption interface {
	isJSON()
}
type SQLOption interface {
	isSQL()
}

type TextOption interface {
	isText()
}

type YAMLOption interface {
	isYAML()
}

func IgnoreMapKeys(keys ...string) cmp.Option {
	return cmpopts.IgnoreMapEntries(func(key string, _ any) bool {
		for _, k := range keys {
			if k == key {
				return true
			}
		}

		return false
	})
}
