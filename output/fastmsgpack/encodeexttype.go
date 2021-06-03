package fastmsgpack

import (
	"github.com/vmihailenco/msgpack/v4/codes"
)

// EncodeExtHeader encodes the header of an extension-typed value
func EncodeExtHeader(buffer []byte, start int, typeID byte, length int) int { // xx:inline
	start = EncodeExtLen(buffer, start, length)
	buffer[start] = typeID
	return start + 1
}

// EncodeExtLen encodes the length of extension-typed value, followed by the extendion type ID
func EncodeExtLen(buffer []byte, start int, length int) int { // xx:inline
	switch length {
	case 1:
		buffer[start] = byte(codes.FixExt1)
	case 2:
		buffer[start] = byte(codes.FixExt2)
	case 4:
		buffer[start] = byte(codes.FixExt4)
	case 8:
		buffer[start] = byte(codes.FixExt8)
	case 16:
		buffer[start] = byte(codes.FixExt16)
	default:
		panic(length)
	}
	return start + 1
}
