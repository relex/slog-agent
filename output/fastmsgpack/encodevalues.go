package fastmsgpack

import (
	"math"

	"github.com/vmihailenco/msgpack/v4/codes"
)

// EncodeFloat64 encodes a 64 bits floating number
func EncodeFloat64(buffer []byte, start int, value float64) int { // can't inline, cost too much
	buffer[start] = byte(codes.Double)
	return Write8(buffer, start+1, math.Float64bits(value))
}

// EncodeInt32 encodes a 32 bits uint
func EncodeInt32(buffer []byte, start int, value int32) int { // xx:inline
	buffer[start] = byte(codes.Int32)
	return Write4(buffer, start+1, uint32(value))
}
