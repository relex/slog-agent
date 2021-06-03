package tredactemail

import (
	"strings"

	"github.com/relex/slog-agent/util"
)

var redactEmailValidAddressChars = make([]bool, 256)
var redactEmailValidNameChars = make([]bool, 256)

func init() {
	for c := byte('A'); c <= byte('Z'); c++ {
		redactEmailValidAddressChars[c] = true
		redactEmailValidNameChars[c] = true
	}
	for c := byte('a'); c <= byte('z'); c++ {
		redactEmailValidAddressChars[c] = true
		redactEmailValidNameChars[c] = true
	}
	for c := byte('0'); c <= byte('9'); c++ {
		redactEmailValidAddressChars[c] = true
		redactEmailValidNameChars[c] = true
	}
	redactEmailValidAddressChars['.'] = true
	redactEmailValidAddressChars['-'] = true
	redactEmailValidAddressChars['_'] = true
}

func redactEmail(src string) string {
	first := redactEmailFindFirst(src)
	if first == -1 {
		return src
	}
	dest, _ := redactEmail1(src, first)
	return dest
}

func redactEmailFindFirst(src string) int {
	sEnd := len(src) - 1
	sAt := strings.IndexByte(src, '@')
	// ignore src[0] and src[len-1] because no valid email possible
	for sAt < sEnd {
		if sAt > 0 && redactEmailValidNameChars[src[sAt-1]] && redactEmailValidNameChars[src[sAt+1]] {
			return sAt
		}
		sAt++
		nextAt := strings.IndexByte(src[sAt:], '@')
		if nextAt == -1 {
			break
		} else {
			sAt += nextAt
		}
	}
	return -1
}

func redactEmail1(src string, start int) (string, int) {
	numRedacted := 0
	dst := make([]byte, 0, len(src))
	sEnd := len(src) - 1
	sAt := start // sAt should point to '@'
	sCopied := 0
	// ignore src[0] and src[len-1] because no valid email possible
	for sAt < sEnd {
		if sAt > 0 && redactEmailValidNameChars[src[sAt-1]] && redactEmailValidNameChars[src[sAt+1]] {
			emailStart := redactFindEmailStart(src, sAt, sCopied)
			emailEnd := redactFindEmailEnd(src, sAt)
			if emailEnd != -1 {
				// copy contents before email and the email
				dst = append(dst, src[sCopied:emailStart]...)
				dst = append(dst, "REDACTED"...)
				sCopied = emailEnd
				sAt = emailEnd
				numRedacted++
			} else {
				sAt++
			}
			if sAt > sEnd {
				break
			}
		} else {
			sAt++
		}
		nextAt := strings.IndexByte(src[sAt:], '@')
		if nextAt == -1 {
			break
		} else {
			sAt += nextAt
		}
	}
	dst = append(dst, src[sCopied:]...)
	return util.StringFromBytes(dst), numRedacted
}

func redactFindEmailStart(src string, atIndex int, limitStart int) int {
	var i int
	for i = atIndex - 1; i >= limitStart; i-- {
		c := src[i]
		if !redactEmailValidAddressChars[c] {
			break
		}
	}
	return i + 1
}

func redactFindEmailEnd(src string, atIndex int) int {
	var i int
	dotIndex := -1
	for i = atIndex + 1; i < len(src); i++ {
		c := src[i]
		if !redactEmailValidAddressChars[c] {
			return -1
		}
		if c == '.' {
			dotIndex = i
			break
		}
	}
	switch {
	case dotIndex == -1:
		// truncated domain, e.g.: foo.bar@google
		if redactEmailCheckNumber(src[atIndex+1:]) {
			return -1
		}
		return len(src)
	case dotIndex == len(src)-1:
		// truncated domain, e.g.: foo.bar@google.
		return len(src)
	case !redactEmailValidNameChars[src[dotIndex+1]]:
		// not email, e.g.: Trx@123456./
		return -1
	}
	for i = dotIndex + 2; i < len(src); i++ {
		c := src[i]
		if !redactEmailValidAddressChars[c] {
			break
		}
	}
	if redactEmailCheckNumber(src[atIndex+1 : i]) {
		return -1
	}
	return i
}

func redactEmailCheckNumber(s string) bool {
	if len(s) < 2 {
		return false
	}
	if first := s[0]; first < '0' || first > '9' {
		return false
	}
	if last := s[len(s)-1]; last < '0' || last > '9' {
		return false
	}
	return true
}
