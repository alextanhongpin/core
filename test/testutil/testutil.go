package testutil

import "github.com/google/go-cmp/cmp"

// Those with x is shared.
type FileName string

func (x FileName) isJSON() {}
func (x FileName) isYAML() {}
func (x FileName) isSQL()  {}
func (x FileName) isGRPC() {}
func (x FileName) isHTTP() {}
func (x FileName) isText() {}

type CmpOption []cmp.Option

func (x CmpOption) isJSON() {}
func (x CmpOption) isYAML() {}

func CmpOpts(opts ...cmp.Option) CmpOption {
	return CmpOption(opts)
}

type ignoreFields []string

func (ignoreFields) isJSON() {}
func (ignoreFields) isYAML() {}

func IgnoreFields(fields ...string) ignoreFields {
	return ignoreFields(fields)
}

type maskFields []string

func (maskFields) isJSON() {}
func (maskFields) isYAML() {}

func MaskFields(fields ...string) maskFields {
	return maskFields(fields)
}
