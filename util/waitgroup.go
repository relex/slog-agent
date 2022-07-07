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
