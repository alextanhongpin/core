package testutil

import (
	"github.com/alextanhongpin/core/types/maputil"
	"github.com/google/go-cmp/cmp"
)

type JSONCmpOptions []cmp.Option

func JSONCmpOption(opts ...cmp.Option) JSONCmpOptions {
	return JSONCmpOptions(opts)
}

func IgnoreFields(fields ...string) JSONCmpOptions {
	return JSONCmpOptions([]cmp.Option{IgnoreMapKeys(fields...)})
}
func (o JSONCmpOptions) isJSON() {}

type JSONInspector func([]byte) error

func (i JSONInspector) isJSON() {}

func InspectJSON(fn func([]byte) error) JSONInspector {
	return fn
}

type JSONInterceptor func([]byte) ([]byte, error)

func (i JSONInterceptor) isJSON() {}

func MaskJSON(fields ...string) JSONInterceptor {
	return func(b []byte) ([]byte, error) {
		return maputil.MaskBytes(b, fields...)
	}
}
