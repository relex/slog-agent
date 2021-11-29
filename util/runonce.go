package util

import (
	"sync/atomic"
)

// RunOnce is a function wrapper that calls the underlying function at most once
//
// beforeRunning is invoked right before the underlying function is invoked and only if it's going to be invoked.
// It may be nil.
//
// This can be used to protect e.g. resource closing or cleanup, which should be called exactly once
type RunOnce func(beforeRunning func())

// NewRunOnce creates a function that would call the given "f" at most once
func NewRunOnce(f func()) RunOnce {
	var invoked int32 = 0
	return func(beforeRunning func()) {
		if atomic.CompareAndSwapInt32(&invoked, 0, 1) {
			if beforeRunning != nil {
				beforeRunning()
			}
			f()
		}
	}
}
