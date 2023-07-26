package util

import (
	"sync"
	"sync/atomic"
)

type TrackedWaitGroup struct {
	wg    sync.WaitGroup
	count int64
}

func (twg *TrackedWaitGroup) Add(delta int) {
	twg.wg.Add(delta)
	atomic.AddInt64(&twg.count, int64(delta))
}

func (twg *TrackedWaitGroup) Done() {
	twg.wg.Done()
	atomic.AddInt64(&twg.count, -1)
}

func (twg *TrackedWaitGroup) Peek() int {
	return int(twg.count)
}

func (twg *TrackedWaitGroup) Wait() {
	twg.wg.Wait()
}
