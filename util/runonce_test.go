package util

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunOnce(t *testing.T) {
	v1 := int64(0)
	v2 := int64(100)

	f := NewRunOnce(func() {
		atomic.AddInt64(&v1, 1)
	})

	before := func() {
		atomic.AddInt64(&v2, 1)
	}

	for i := 0; i < 1000; i++ {
		go func() {
			f(before)
		}()
	}
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int64(1), v1)
	assert.Equal(t, int64(101), v2)
}
