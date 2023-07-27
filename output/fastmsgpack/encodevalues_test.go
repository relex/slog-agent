package fastmsgpack

import (
	"bytes"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v4"
)

func TestEncodeFloat64(t *testing.T) {
	for _, floatValue := range []float64{0, 0.5, math.MaxFloat64, float64(math.MinInt64)} {
		buf := make([]byte, 100)
		off := EncodeFloat64(buf, 0, floatValue)
		dlen, derr := msgpack.NewDecoder(bytes.NewBuffer(buf[:off])).DecodeFloat64()
		assert.NoError(t, derr, "test encoding float64 %f", floatValue)
		assert.Equal(t, floatValue, dlen, "test encoding float64 %f", floatValue)
	}
}

func TestEncodeInt32(t *testing.T) {
	for _, intValue := range []int32{10, math.MaxInt32, math.MinInt32} {
		buf := make([]byte, 100)
		off := EncodeInt32(buf, 0, intValue)
		dlen, derr := msgpack.NewDecoder(bytes.NewBuffer(buf[:off])).DecodeInt32()
		assert.NoError(t, derr, "test encoding int32 %f", intValue)
		assert.Equal(t, intValue, dlen, "test encoding int32 %f", intValue)
	}
}
