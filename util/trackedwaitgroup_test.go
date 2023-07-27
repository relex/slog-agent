package util

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrackedWaitGroup(t *testing.T) {
	twg := &TrackedWaitGroup{}
	assert.Equal(t, 0, twg.Peek())
	twg.Add(1)
	twg.Add(1)
	assert.Equal(t, 2, twg.Peek())
	twg.Done()
	assert.Equal(t, 1, twg.Peek())

	var doneCalled int64
	go func() {
		twg.Done()
		atomic.AddInt64(&doneCalled, 1)
	}()

	twg.Wait()
	assert.Equal(t, int64(1), atomic.LoadInt64(&doneCalled))
}
