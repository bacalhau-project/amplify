package util

import (
	"sort"
)

// contains is a helper function that iterates over a slice and returns true if the given value is found
func Contains[T comparable](slice []T, value T) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// Dedup removes duplicate strings from a slice
func Dedup[T comparable](s []T) []T {
	m := make(map[T]bool)
	for _, v := range s {
		m[v] = true
	}
	var results []T
	for k := range m {
		results = append(results, k)
	}
	return results
}

func SortSliceInt32(s []int32) {
	sort.Slice(s, func(i, j int) bool {
		return s[i] < s[j]
	})
}
