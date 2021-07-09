package fastmsgpack

import (
	"github.com/vmihailenco/msgpack/v4/codes"
)

func EncodeExtHeader8(buffer []byte, start int, typeID byte) int { // xx:inline
	buffer[start] = byte(codes.FixExt8)
	buffer[start+1] = typeID
	return start + 2
}
