package util

import (
	"unsafe"
)

// DeepCopyString copies the given string to a newly-allocated one
//
// Without references to the original backing bytes
func DeepCopyString(str string) string {
	// Force creation of new backing byte-array by converting to mutable []byte first
	// See https://stackoverflow.com/a/35993927/3488757
	return StringFromBytes([]byte(str))
}

// DeepCopyStringFromBytes copies the given []byte to a newly-allocated string
//
// Without references to the original backing bytes
func DeepCopyStringFromBytes(str []byte) string {
	// no other action needed; Go forces the copy internally as []byte is mutable but bytes used by string isn't
	// hence []byte to []byte change or substring (= take a slice) performs no copy
	// but string from []byte always does
	return string(str)
}

// DeepCopyStrings copies the given string list including each of item to newly allocated fields
//
// Without references to the original backing slices
func DeepCopyStrings(strList []string) []string {
	destList := make([]string, len(strList))
	for i, str := range strList {
		destList[i] = DeepCopyString(str)
	}
	return destList
}

// StringFromBytes makes a string pointing to the contents of []byte
//
// There is no copying and the resulting string shares the same []byte contents
//
// If data in the backing byte array is changed, the string contents would reflect the changes (NOT normal Go string behavior).
//
// DO NOT use this in tests
func StringFromBytes(buf []byte) string {
	// code from strings.Builder.String()
	// GO_INTERNAL
	// This works because reflect.StringHeader is identical to the front part of reflect.SliceHeader
	return *(*string)(unsafe.Pointer(&buf))
}
