package list_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/alextanhongpin/core/types/list"
	"github.com/go-openapi/testify/assert"
)

func TestMap(t *testing.T) {
	is := assert.New(t)
	is.Equal([]string{"1", "2"}, list.Map([]int{1, 2}, strconv.Itoa))
	is.Nil(list.Map[int, string](nil, strconv.Itoa))
	is.Equal([]string{}, list.Map([]int{}, strconv.Itoa))
}

func TestMapError(t *testing.T) {
	is := assert.New(t)
	got, err := list.MapError([]string{"1", "2"}, strconv.Atoi)
	is.NoError(err)
	is.Equal([]int{1, 2}, got)

	calls := 0
	got, err = list.MapError([]string{"1", "x", "3"}, func(s string) (int, error) {
		calls++
		return strconv.Atoi(s)
	})
	is.Error(err)
	is.Nil(got)
	is.Equal(2, calls, "stops at the first error")
}

func TestFilter(t *testing.T) {
	is := assert.New(t)
	even := func(n int) bool { return n%2 == 0 }
	in := []int{1, 2, 3, 4}
	is.Equal([]int{2, 4}, list.Filter(in, even))
	is.Equal([]int{1, 2, 3, 4}, in, "input is not modified")
	is.Nil(list.Filter([]int{1, 3}, even))
}

func TestFlatMap(t *testing.T) {
	got := list.FlatMap([]int{1, 2, 3}, func(n int) []int { return []int{n, n} })
	assert.Equal(t, []int{1, 1, 2, 2, 3, 3}, got)
}

func TestReduce(t *testing.T) {
	got := list.Reduce([]int{1, 2, 3}, "", func(acc string, n int) string {
		return acc + strconv.Itoa(n)
	})
	assert.Equal(t, "123", got)
	assert.Equal(t, 10, list.Reduce([]int(nil), 10, func(acc, n int) int { return acc + n }))
}

func TestGroupBy(t *testing.T) {
	got := list.GroupBy([]string{"apple", "bob", "avocado"}, func(s string) byte { return s[0] })
	assert.Equal(t, map[byte][]string{'a': {"apple", "avocado"}, 'b': {"bob"}}, got)
}

func TestPartition(t *testing.T) {
	is := assert.New(t)
	matched, rest := list.Partition([]int{1, 2, 3, 4, 5}, func(n int) bool { return n > 2 })
	is.Equal([]int{3, 4, 5}, matched)
	is.Equal([]int{1, 2}, rest)
}

func TestUniqueFunc(t *testing.T) {
	type user struct {
		ID   int
		Name string
	}
	in := []user{{1, "a"}, {2, "b"}, {1, "c"}}
	got := list.UniqueFunc(in, func(u user) int { return u.ID })
	assert.Equal(t, []user{{1, "a"}, {2, "b"}}, got)
	assert.Nil(t, list.UniqueFunc[user](nil, func(u user) int { return u.ID }))
}

func TestSum(t *testing.T) {
	assert.Equal(t, 6, list.Sum([]int{1, 2, 3}))
	assert.Equal(t, 0.0, list.Sum([]float64{}))
}

func ExampleMapError() {
	nums, err := list.MapError([]string{"1", "2", "x"}, strconv.Atoi)
	fmt.Println(nums, errors.Is(err, strconv.ErrSyntax))
	// Output: [] true
}

func ExampleGroupBy() {
	byLen := list.GroupBy([]string{"go", "rust", "c", "zig"}, func(s string) int { return len(s) })
	fmt.Println(byLen[1], byLen[2], byLen[3], byLen[4])
	// Output: [c] [go] [zig] [rust]
}
