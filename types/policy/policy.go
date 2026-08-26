package policy

import "slices"

type Policy struct {
	AllowList []string
	DenyList  []string
}

func (p Policy) Allow(val string) bool {
	if slices.Contains(p.DenyList, val) {
		return false
	}
	if slices.Contains(p.AllowList, val) {
		return true
	}

	// If empty, default allow, else default deny.
	return len(p.AllowList) == 0
}
