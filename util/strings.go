package util

import (
	"strings"
	"unsafe"
)

// CompareStrings compares the given strings and returns <0 if s1 < s2, >0 if s1 > s2, and zero if equal
func CompareStrings(s1 []string, s2 []string) int {
	clen := MinInt(len(s1), len(s2))

	for i := 0; i < clen; i++ {
		if c := strings.Compare(s1[i], s2[i]); c != 0 {
			return c
		}
	}

	return len(s1) - len(s2)
}

// ContainsString checks whether target is inside the given list
func ContainsString(list []string, target string) bool {
	return Any(len(list), func(i int) bool { return list[i] == target })
}

// DeepCopyString copies the given string to a newly-allocated one
//
// Without references to the original backing bytes
func DeepCopyString(str string) string {
	// Force new backing string by converting to mutable []byte first
	// See https://stackoverflow.com/a/35993927/3488757
	return StringFromBytes([]byte(str))
}

// DeepCopyStringFromBytes copies the given []byte to a newly-allocated string
//
// Without references to the original backing bytes
func DeepCopyStringFromBytes(str []byte) string {
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

// MapString produces a new list by mapping given list using specified function
//
//     lower := []string{"bob", "david", "april"}
//     upper := MapString(lower, strings.ToUpper)
func MapString(input []string, mapFunc func(in string) string) []string {
	output := make([]string, len(input))
	for i := 0; i < len(input); i++ {
		output[i] = mapFunc(input[i])
	}
	return output
}

// StringFromBytes makes a string pointing to the contents of []byte
//
// There is no copying and the resulting string shares the same []byte contents
//
// DO NOT use this in tests
func StringFromBytes(buf []byte) string {
	// code from strings.Builder.String()
	// This works because reflect.StringHeader is identical to the front part of reflect.SliceHeader
	// GO_INTERNAL
	return *(*string)(unsafe.Pointer(&buf))
}
