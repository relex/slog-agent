package fastmsgpack

import (
	"github.com/vmihailenco/msgpack/v4/codes"
)

// EncodeExtHeader8 encodes an extension type of 8 bytes
func EncodeExtHeader8(buffer []byte, start int, typeID byte) int { // xx:inline
	buffer[start] = byte(codes.FixExt8)
	buffer[start+1] = typeID
	return start + 2
}
