package util

import (
	"golang.org/x/exp/constraints"
)

// CopySlice copies the slice to a newly-allocated one
func CopySlice[T any](slice []T) []T { // xx:inline
	return append([]T(nil), slice...)
}

// EachInSlice calls the given func for each of (index, value) pair in the given slice
func EachInSlice[T any](slice []T, action func(index int, item T)) {
	for index, item := range slice {
		action(index, item)
	}
}

// MapSlice transforms the given slice by mapping each item to something else
func MapSlice[T any, R any](slice []T, mapper func(item T) R) []R {
	output := make([]R, len(slice))
	for index, item := range slice {
		output[index] = mapper(item)
	}
	return output
}

// SumSlice sums up the values calculated from each item in the given slice
func SumSlice[T any, R constraints.Integer | constraints.Float](slice []T, calculate func(item T) R) R {
	var result R
	for _, item := range slice {
		result += calculate(item)
	}
	return result
}
