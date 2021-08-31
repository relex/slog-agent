package util

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

const mutexLocked = 1 << iota // mutex is locked

// _mutex is the unexposed definition of sync.Mutex
//
// GO_INTERNAL
type _mutex struct {
	state int32
	// sema  uint32
}

// TryLockMutex attempts to lock mutex or fail
//
// FIXME: this may be implemented in Go 1.18, see https://github.com/golang/go/issues/45435
func TryLockMutex(mutex *sync.Mutex) bool {
	m := (*_mutex)(unsafe.Pointer(mutex))
	return atomic.CompareAndSwapInt32(&m.state, 0, mutexLocked)
}
