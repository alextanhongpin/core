package internal

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func IgnoreMapEntries(keys ...string) cmp.Option {
	return cmpopts.IgnoreMapEntries(func(k string, v any) bool {
		for _, key := range keys {
			if key == k {
				return true
			}
		}

		return false
	})
}
