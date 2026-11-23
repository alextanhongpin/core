// Package list provides generic slice helpers that the standard library
// slices and maps packages do not: mapping, filtering, reducing, grouping,
// partitioning, and key-based deduplication.
//
// Every function takes and returns plain slices and never modifies its input.
package list

import (
	"golang.org/x/exp/constraints"
)

// Map returns a new slice holding fn applied to each element of s, in order.
// It returns nil when s is nil.
func Map[T, U any](s []T, fn func(T) U) []U {
	if s == nil {
		return nil
	}
	out := make([]U, len(s))
	for i, v := range s {
		out[i] = fn(v)
	}
	return out
}

// MapError is like Map, but stops at the first error and returns it with a
// nil slice.
func MapError[T, U any](s []T, fn func(T) (U, error)) ([]U, error) {
	if s == nil {
		return nil, nil
	}
	out := make([]U, len(s))
	for i, v := range s {
		u, err := fn(v)
		if err != nil {
			return nil, err
		}
		out[i] = u
	}
	return out, nil
}

// Filter returns a new slice holding the elements of s for which keep
// returns true, in order. It returns nil when no element is kept.
func Filter[T any](s []T, keep func(T) bool) []T {
	var out []T
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// FlatMap applies fn to each element of s and concatenates the results.
func FlatMap[T, U any](s []T, fn func(T) []U) []U {
	var out []U
	for _, v := range s {
		out = append(out, fn(v)...)
	}
	return out
}

// Reduce folds s from left to right, starting from initial.
func Reduce[T, U any](s []T, initial U, fn func(acc U, v T) U) U {
	acc := initial
	for _, v := range s {
		acc = fn(acc, v)
	}
	return acc
}

// GroupBy groups the elements of s by key. Elements keep their original
// relative order within each group.
func GroupBy[T any, K comparable](s []T, key func(T) K) map[K][]T {
	groups := make(map[K][]T)
	for _, v := range s {
		k := key(v)
		groups[k] = append(groups[k], v)
	}
	return groups
}

// Partition splits s into the elements for which pred returns true and those
// for which it returns false, preserving order in both.
func Partition[T any](s []T, pred func(T) bool) (matched, rest []T) {
	for _, v := range s {
		if pred(v) {
			matched = append(matched, v)
		} else {
			rest = append(rest, v)
		}
	}
	return matched, rest
}

// UniqueFunc returns the elements of s with duplicate keys removed, keeping
// the first element seen for each key. For comparable elements use
// sets.Unique, or slices.Compact on sorted input.
func UniqueFunc[T any, K comparable](s []T, key func(T) K) []T {
	if s == nil {
		return nil
	}
	seen := make(map[K]struct{}, len(s))
	out := make([]T, 0, len(s))
	for _, v := range s {
		k := key(v)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, v)
	}
	return out
}

// Sum returns the sum of s, or zero for an empty slice. Integer sums wrap on
// overflow.
func Sum[T constraints.Integer | constraints.Float](s []T) T {
	var total T
	for _, v := range s {
		total += v
	}
	return total
}
