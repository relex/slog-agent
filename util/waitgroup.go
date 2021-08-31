package util

import (
	"sync"
	"unsafe"
)

// _waitGroup is the unexposed definition of sync.WaitGroup
//
// GO_INTERNAL
type _waitGroup struct {
	NoCopy struct{}

	// 64-bit value: high 32 bits are counter, low 32 bits are waiter count.
	// 64-bit atomic operations require 64-bit alignment, but 32-bit
	// compilers do not ensure it. So we allocate 12 bytes and then use
	// the aligned 8 bytes in them as state, and the other 4 as storage
	// for the sema.
	state1 [3]uint32
}

// PeekWaitGroup returns the count from the internal counter of sync.WaitGroup by a non-atomic op
func PeekWaitGroup(waitGroup *sync.WaitGroup) int {
	wg := (*_waitGroup)(unsafe.Pointer(waitGroup))
	statep, _ := wg.state()
	return int(*statep >> 32)
}

// state is a copy of sync.WaitGroup.state()
func (wg *_waitGroup) state() (statep *uint64, semap *uint32) {
	if uintptr(unsafe.Pointer(&wg.state1))%8 == 0 {
		return (*uint64)(unsafe.Pointer(&wg.state1)), &wg.state1[2]
	}
	return (*uint64)(unsafe.Pointer(&wg.state1[1])), &wg.state1[0]
}
