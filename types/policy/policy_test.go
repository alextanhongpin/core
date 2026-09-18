package policy_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/policy"
	"github.com/alextanhongpin/evaltest"
)

func TestPolicy(t *testing.T) {
	type Input struct {
		Value     string   `json:"value"`
		AllowList []string `json:"allowlist,omitempty"`
		DenyList  []string `json:"denylist,omitempty"`
	}

	evaltest.Run(t, func(t *evaltest.T, input Input) (bool, error) {
		p := policy.Policy{AllowList: input.AllowList, DenyList: input.DenyList}
		return p.Allow(input.Value), nil
	})
}
