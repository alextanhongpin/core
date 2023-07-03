package sliceutil_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/sliceutil"
	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	assert := assert.New(t)
	nums := []int{1, 2, 3, 4, 5}

	nums2x := sliceutil.Map(nums, func(i int) int {
		return nums[i] * 2
	})

	assert.Equal([]int{2, 4, 6, 8, 10}, nums2x)
}

func TestFilter(t *testing.T) {
	assert := assert.New(t)
	nums := []int{1, 2, 3, 4, 5}

	oddNums := sliceutil.Filter(nums, func(i int) bool {
		return nums[i]%2 == 1
	})

	assert.Equal([]int{1, 3, 5}, oddNums)
}

func TestSum(t *testing.T) {
	assert := assert.New(t)
	nums := []int{1, 2, 3, 4, 5}

	total := sliceutil.Sum(nums)
	assert.Equal(15, total)
}

func TestMin(t *testing.T) {
	assert := assert.New(t)
	nums := []int{-100, 1, 2, 3, 4, 5}

	total := sliceutil.Min(nums)
	assert.Equal(-100, total)
}

func TestMax(t *testing.T) {
	assert := assert.New(t)
	nums := []int{-100, 1, 2, 3, 4, 5}

	total := sliceutil.Max(nums)
	assert.Equal(5, total)
}
