package testutil

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

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
