package fastmsgpack

import (
	"github.com/vmihailenco/msgpack/v4/codes"
)

// EncodeArrayLen4 encodes the header of array up to 15 items
func EncodeArrayLen4(buffer []byte, start int, arrayLen int) int { // xx:inline
	buffer[start] = byte(codes.FixedArrayLow) | byte(arrayLen)
	return start + 1
}

// EncodeArrayLen16 encodes the header of array up to 65535 items
func EncodeArrayLen16(buffer []byte, start int, arrayLen int) int { // xx:inline
	buffer[start] = byte(codes.Array16)
	return Write2(buffer, start+1, uint16(arrayLen))
}

// EncodeArrayLen32 encodes the header of array larger than 65535 items
func EncodeArrayLen32(buffer []byte, start int, arrayLen int) int { // xx:inline
	buffer[start] = byte(codes.Array32)
	return Write4(buffer, start+1, uint32(arrayLen))
}

// EncodeMapLen4 encodes the header of map up to 15 items
func EncodeMapLen4(buffer []byte, start int, mapLen int) int { // xx:inline
	buffer[start] = byte(codes.FixedMapLow) | byte(mapLen)
	return start + 1
}

// EncodeMapLen16 encodes the header of map up to 65535 items
func EncodeMapLen16(buffer []byte, start int, mapLen int) int { // xx:inline
	buffer[start] = byte(codes.Map16)
	return Write2(buffer, start+1, uint16(mapLen))
}

// EncodeMapLen32 encodes the header of map larger than 65535 items
func EncodeMapLen32(buffer []byte, start int, mapLen int) int { // xx:inline
	buffer[start] = byte(codes.Map32)
	return Write4(buffer, start+1, uint32(mapLen))
}

// EncodeString4 encodes string up to 15 bytes
func EncodeString4(buffer []byte, start int, str string) int { // xx:inline
	pos := EncodeStringLen4(buffer, start, len(str))
	pos += copy(buffer[pos:], str)
	return pos
}

// EncodeString16 encodes string up to 65535 bytes
func EncodeString16(buffer []byte, start int, str string) int { // xx:inline
	pos := EncodeStringLen16(buffer, start, len(str))
	pos += copy(buffer[pos:], str)
	return pos
}

// EncodeString32 encodes string larger than 65535 bytes
func EncodeString32(buffer []byte, start int, str string) int { // xx:inline
	pos := EncodeStringLen32(buffer, start, len(str))
	pos += copy(buffer[pos:], str)
	return pos
}

// EncodeStringLen4 encodes the header of string up to 15 bytes
func EncodeStringLen4(buffer []byte, start int, strLen int) int { // xx:inline
	buffer[start] = byte(codes.FixedStrLow) | byte(strLen)
	return start + 1
}

// EncodeStringLen16 encodes the header of string up to 65535 bytes
func EncodeStringLen16(buffer []byte, start int, strLen int) int { // xx:inline
	buffer[start] = byte(codes.Str16)
	return Write2(buffer, start+1, uint16(strLen))
}

// EncodeStringLen32 encodes the header of string larger than 65535 bytes
func EncodeStringLen32(buffer []byte, start int, strLen int) int { // xx:inline
	buffer[start] = byte(codes.Str32)
	return Write4(buffer, start+1, uint32(strLen))
}

// ReserveLen4 reserves space to encode length of some type up to 15 items
func ReserveLen4(start int) int { // xx:inline
	return start + 1
}

// ReserveLen16 reserves space to encode length of some type up to 65535 items
func ReserveLen16(start int) int { // xx:inline
	return start + 3
}

// ReserveLen32 reserves space to encode length of some type more than 65535 items
func ReserveLen32(start int) int { // xx:inline
	return start + 5
}
