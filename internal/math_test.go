package internal_test

import (
	"testing"

	"github.com/alextanhongpin/core/internal"
	"github.com/stretchr/testify/assert"
)

func TestGCD(t *testing.T) {
	tests := []struct {
		name string
		args []int64
		want int64
	}{
		{
			name: "increasing number",
			args: []int64{1, 2, 3, 4, 5},
			want: 1,
		},
		{
			name: "division of 5",
			args: []int64{5, 10, 25, 100},
			want: 5,
		},
		{
			name: "zero",
			args: []int64{},
			want: 0,
		},
		{
			name: "one",
			args: []int64{1},
			want: 1,
		},
		{
			name: "all equal",
			args: []int64{1, 1, 1, 1, 1},
			want: 1,
		},
		{
			name: "prime number",
			args: []int64{3, 7, 11},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			got := internal.GCD(tt.args)
			assert.Equal(tt.want, got)
		})
	}
}
