package util

import (
	"math/bits"
	"sync"
)

// BytesPoolBy2n is a set of pools by size
//
// BytesPoolBy2n[1] = pool of make([]byte, 2)
// BytesPoolBy2n[2] = pool of make([]byte, 4)
// BytesPoolBy2n[8] = pool of make([]byte, 256)
// ...
// Local-cache with buffers of recycled []byte has been tried and made minimal improvement
type BytesPoolBy2n []*sync.Pool

// NewBytesPoolBy2n creates a BytesPoolBy2n with all pools initialized
func NewBytesPoolBy2n() BytesPoolBy2n {
	poolBy2n := make(BytesPoolBy2n, 32)
	for n := range poolBy2n {
		size := 1 << n
		poolBy2n[n] = &sync.Pool{
			New: func() interface{} {
				b := make([]byte, size)
				return &b
			},
		}
	}
	return poolBy2n
}

// Get fetches an empty byte array suitable for given length of data
// The array returned may be longer than the given length
func (pools BytesPoolBy2n) Get(length int) *[]byte {
	capacity := 32 - bits.LeadingZeros32(uint32(length))
	buf := pools[capacity].Get().(*[]byte)
	return buf
}

// Put recycles the given byte array into pool
// The array specified must come from BytesPoolBy2n.Get()
func (pools BytesPoolBy2n) Put(buf *[]byte) {
	length := len(*buf)
	capacity := 32 - bits.LeadingZeros32(uint32(length)) - 1
	pools[capacity].Put(buf)
}
