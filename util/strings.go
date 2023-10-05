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

// MutableString is a string backed by raw []byte, instead of in the immutable memory area like normal Go strings.
//
// Its contents may be changed. But we cannot create a new type or string functions wouldn't work with it.
type MutableString = string

// StringFromBytes makes a string backed by a specified []byte.
//
// There is no copying and the resulting string shares the same []byte contents.
//
// If data in the backing slice is changed, the string contents would reflect the changes (NOT normal Go string behavior).
//
// DO NOT use this in tests.
func StringFromBytes(buf []byte) MutableString {
	// code from strings.Builder.String()
	return unsafe.String(unsafe.SliceData(buf), len(buf))
}

// BytesFromString makes a []byte pointing to the contents of a string.
//
// The string must come from StringFromBytes if the new []byte is to be modified, as normal Go strings have their data
// allocated in the immutable memory area and any write operation would trigger panics.
func BytesFromString(str MutableString) []byte {
	return unsafe.Slice(unsafe.StringData(str), len(str))
}
