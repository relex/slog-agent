package fastmsgpack

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v4"
)

func TestEncodeArrayLen(t *testing.T) {
	buf := make([]byte, 1000)
	start := 0
	end := 0
	decoder := msgpack.NewDecoder(bytes.NewBuffer(buf))
	msg4 := "test encoding array4 len="
	for _, lenValue := range []int{2, 15} {
		end = EncodeArrayLen4(buf, start, lenValue)
		assert.Equal(t, 1, end-start, msg4, lenValue)
		dlen, derr := decoder.DecodeArrayLen()
		assert.NoError(t, derr, msg4, lenValue)
		assert.Equal(t, lenValue, dlen, msg4, lenValue)
		start = end
	}
	msg16 := "test encoding array16 len="
	for _, lenValue := range []int{2, 32768, 65535} {
		end = EncodeArrayLen16(buf, start, lenValue)
		assert.Equal(t, 3, end-start, msg16, lenValue)
		dlen, derr := decoder.DecodeArrayLen()
		assert.NoError(t, derr, msg16, lenValue)
		assert.Equal(t, lenValue, dlen, msg16, lenValue)
		start = end
	}
	msg32 := "test encoding array32 len="
	for _, lenValue := range []int{2, 32768, 65536, 16777216} {
		end = EncodeArrayLen32(buf, start, lenValue)
		assert.Equal(t, 5, end-start, msg32, lenValue)
		dlen, derr := decoder.DecodeArrayLen()
		assert.NoError(t, derr, msg32, lenValue)
		assert.Equal(t, lenValue, dlen, msg32, lenValue)
		start = end
	}
}

func TestEncodeMapLen(t *testing.T) {
	msg4 := "test encoding map4 len="
	for _, lenValue := range []int{2, 15} {
		buf := make([]byte, 100)
		pos := EncodeMapLen4(buf, 0, lenValue)
		assert.Equal(t, 1, pos, msg4, lenValue)
		dlen, derr := msgpack.NewDecoder(bytes.NewBuffer(buf[:pos])).DecodeMapLen()
		assert.NoError(t, derr, msg4, lenValue)
		assert.Equal(t, lenValue, dlen, msg4, lenValue)
	}
	msg16 := "test encoding map16 len="
	for _, lenValue := range []int{2, 32768, 65535} {
		buf := make([]byte, 100)
		pos := EncodeMapLen16(buf, 0, lenValue)
		assert.Equal(t, 3, pos, msg16, lenValue)
		dlen, derr := msgpack.NewDecoder(bytes.NewBuffer(buf[:pos])).DecodeMapLen()
		assert.NoError(t, derr, msg16, lenValue)
		assert.Equal(t, lenValue, dlen, msg16, lenValue)
	}
	msg32 := "test encoding map32 len="
	for _, lenValue := range []int{2, 32768, 65536, 16777216} {
		buf := make([]byte, 100)
		pos := EncodeMapLen32(buf, 0, lenValue)
		assert.Equal(t, 5, pos, msg32, lenValue)
		dlen, derr := msgpack.NewDecoder(bytes.NewBuffer(buf[:pos])).DecodeMapLen()
		assert.NoError(t, derr, msg32, lenValue)
		assert.Equal(t, lenValue, dlen, msg32, lenValue)
	}
}

func TestEncodeString(t *testing.T) {
	msg4 := "test encoding str4 ="
	for _, strValue := range []string{"", "helloWorld", "0123456789abcde"} {
		buf := make([]byte, 100)
		pos := EncodeString4(buf, 0, strValue)
		assert.Equal(t, len(strValue)+1, pos, msg4, strValue)
		dstr, derr := msgpack.NewDecoder(bytes.NewBuffer(buf[:pos])).DecodeString()
		assert.NoError(t, derr, msg4, strValue)
		assert.Equal(t, strValue, dstr, msg4, strValue)
	}
	msg16 := "test encoding str16 ="
	for _, strValue := range []string{"", "helloWorld", "012345678901234567890123456789"} {
		buf := make([]byte, 100)
		pos := EncodeString16(buf, 0, strValue)
		assert.Equal(t, len(strValue)+3, pos, msg16, strValue)
		dstr, derr := msgpack.NewDecoder(bytes.NewBuffer(buf[:pos])).DecodeString()
		assert.NoError(t, derr, msg16, strValue)
		assert.Equal(t, strValue, dstr, msg16, strValue)
	}
	msg32 := "test encoding str32 len="
	for _, strValue := range []string{"", "helloWorld", "012345678901234567890123456789", strings.Repeat("0123456789", 10000)} {
		buf := make([]byte, 100100)
		pos := EncodeString32(buf, 0, strValue)
		assert.Equal(t, len(strValue)+5, pos, msg32, len(strValue))
		dstr, derr := msgpack.NewDecoder(bytes.NewBuffer(buf[:pos])).DecodeString()
		assert.NoError(t, derr, msg32, len(strValue))
		assert.Equal(t, strValue, dstr, msg32, len(strValue))
	}
}

func TestEncodeStringOverflow(t *testing.T) {
	buf := make([]byte, 100)
	str := strings.Repeat("0123456789", 10000)
	pos := EncodeString32(buf, 0, str)
	assert.Equal(t, pos, len(buf)) // it's up to caller to check
}
