package testutil

import (
	"github.com/alextanhongpin/core/types/maputil"
	"github.com/google/go-cmp/cmp"
)

type YAMLCmpOptions []cmp.Option

func YAMLCmpOption(opts ...cmp.Option) YAMLCmpOptions {
	return YAMLCmpOptions(opts)
}

func IgnoreKeys(keys ...string) YAMLCmpOptions {
	return YAMLCmpOptions([]cmp.Option{IgnoreMapKeys(keys...)})
}

func (o YAMLCmpOptions) isYAML() {}

type YAMLInspector func([]byte) error

func (i YAMLInspector) isYAML() {}

func InspectYAML(fn func([]byte) error) YAMLInspector {
	return fn
}

type YAMLInterceptor func([]byte) ([]byte, error)

func (i YAMLInterceptor) isYAML() {}

func MaskKeys(keys ...string) YAMLInterceptor {
	return func(b []byte) ([]byte, error) {
		return maputil.MaskBytes(b, keys...)
	}
}
