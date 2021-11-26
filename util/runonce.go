package util

import (
	"sync/atomic"
)

// RunOnce is a function wrapper that calls the underlying function at most once
//
// Returns true when the wrapper function is actually called
//
// This can be used to protect e.g. resource closing or cleanup, which should be called exactly once
type RunOnce func() bool

// NewRunOnce creates a function that would call the given "f" at most once
func NewRunOnce(f func()) func() bool {
	var invoked int32 = 0
	return func() bool {
		if atomic.CompareAndSwapInt32(&invoked, 0, 1) {
			f()
			return true
		}
		return false
	}
}
