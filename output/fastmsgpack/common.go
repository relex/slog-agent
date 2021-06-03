package fastmsgpack

// Write2 encodes big-endian uint of 2 bytes
func Write2(buffer []byte, start int, n uint16) int { // xx:inline
	buffer[start] = byte(n >> 8)
	buffer[start+1] = byte(n)
	return start + 2
}

// Write4 encodes big-endian uint of 4 bytes
func Write4(buffer []byte, start int, n uint32) int { // xx:inline
	buffer[start] = byte(n >> 24)
	buffer[start+1] = byte(n >> 16)
	buffer[start+2] = byte(n >> 8)
	buffer[start+3] = byte(n)
	return start + 4
}

// Write8 encodes big-endian uint of 8 bytes
func Write8(buffer []byte, start int, n uint64) int { // xx:inline
	buffer[start] = byte(n >> 56)
	buffer[start+1] = byte(n >> 48)
	buffer[start+2] = byte(n >> 40)
	buffer[start+3] = byte(n >> 32)
	buffer[start+4] = byte(n >> 24)
	buffer[start+5] = byte(n >> 16)
	buffer[start+6] = byte(n >> 8)
	buffer[start+7] = byte(n)
	return start + 8
}
