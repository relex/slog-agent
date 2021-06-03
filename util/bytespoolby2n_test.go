package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBytesPoolBy2n(t *testing.T) {
	pools := NewBytesPoolBy2n()
	assert.Equal(t, 1, len(*pools[0].New().(*[]byte)))
	assert.Equal(t, 1024, len(*pools[10].New().(*[]byte)))
	if b := pools.Get(255); assert.Equal(t, 256, len(*b)) {
		pools.Put(b)
	}
	// get from all pools again to check the last release is not misplaced
	for n := 0; n < 32; n++ {
		sz := 1 << n
		b := pools.Get(sz - 1)
		assert.Equal(t, sz, len(*b), fmt.Sprintf("released buf was misplaced to pools[%d]", n))
	}
}
