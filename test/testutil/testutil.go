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
