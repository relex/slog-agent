package util

// CopySlice copies the slice to a newly-allocated one
func CopySlice[T any](slice []T) []T { // xx:inline
	return append([]T(nil), slice...)
}
