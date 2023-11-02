package util

import (
	"strings"
)

func CleanUTF8(s []byte) []byte {
	if len(s) == 0 {
		return s
	}
	// correct encoding is https://en.wikipedia.org/wiki/UTF-8
	// too much work; just cut everything after the last ASCII byte
	// FIXME: get a library to clean up invalid sequences without costly copying or moving.
	endPos := findLastEndOfASCII(s)
	uncleanTail := StringFromBytes(s[endPos:])
	return OverwriteNTruncate(s, endPos, strings.ToValidUTF8(uncleanTail, ""))
}

func findLastEndOfASCII(s []byte) int {
	n := len(s)
	for i := n - 1; i >= 0; i-- {
		b := s[i]

		if b <= 0x7F {
			return i + 1
		}
	}
	return 0
}
