package util

import (
	"sync/atomic"
	"unsafe"
)

// AtomicRef is a generic version of atomic.Value, to hold object references atomically
type AtomicRef[T any] struct {
	pointer unsafe.Pointer
}

// Get retrieves the reference atomically. It may return nil.
func (ref *AtomicRef[T]) Get() *T {
	return (*T)(atomic.LoadPointer(&ref.pointer))
}

// Set stores the given reference atomically. The reference may be nil.
func (ref *AtomicRef[T]) Set(reference *T) {
	if reference == nil {
		atomic.StorePointer(&ref.pointer, nil)
	} else {
		atomic.StorePointer(&ref.pointer, unsafe.Pointer(reference))
	}
}
