// Package stringunescape provides Unescaper(s) for escaped strings
package stringunescape

import (
	"fmt"
	"strings"

	"github.com/relex/slog-agent/util"
)

// Unescaper is used to search and unescape characters like '\n', '\t' etc
// Unescaper instances contain no buffer and may be copied or concurrently used.
type Unescaper struct {
	escapeChar       byte
	escapableCharMap []byte
}

// NewUnescaper creates a StringEscaper to be used with map[string]string sources
func NewUnescaper(escapeChar byte, mapping map[byte]byte) Unescaper {
	cmap := make([]byte, 256)
	for key, val := range mapping {
		cmap[key] = val
	}
	cmap[escapeChar] = escapeChar
	return Unescaper{escapeChar, cmap}
}

// FindFirst finds the index of the first escape char, or -1
func (e Unescaper) FindFirst(str string) int {
	return strings.IndexByte(str, e.escapeChar)
}

// FindFirstUnescaped finds the index of the first unescaped target, or -1
// This can be used to find, e.g. the second asterisk in `\*name: Foo*`
func (e Unescaper) FindFirstUnescaped(str string, target byte) int {
	if e.escapableCharMap[target] == 0 {
		panic(fmt.Sprintf("target '%c' is not escapable", target))
	}
	pos := 0
	for pos < len(str) {
		c := str[pos]
		switch c {
		case e.escapeChar:
			pos += 2
		case target:
			return pos
		default:
			pos++
		}
	}
	return -1
}

// Run unescapes the given string
func (e Unescaper) Run(src string) string {
	first := strings.IndexByte(src, e.escapeChar)
	if first == -1 {
		return src
	}
	return e.RunFromFirst(src, first)
}

// RunFromFirst unescapes the given string, starting from the position of first escape char
func (e Unescaper) RunFromFirst(src string, first int) string {
	dst := make([]byte, len(src))
	dend := e.RunToBuffer(src, first, dst)
	return util.StringFromBytes(dst[:dend])
}

// RunToBuffer unescapes the given string to destination buffer, starting from the position of first escape char
// Returns the end / length in the destination buffer
func (e Unescaper) RunToBuffer(src string, first int, dst []byte) int {
	si := first
	di := copy(dst, src[:si])
	slimit := len(src) - 1
	for si < slimit {
		// lookup escape char
		val := src[si+1]
		if c := e.escapableCharMap[val]; c != 0 {
			dst[di] = c
			di++
		} else {
			dst[di] = e.escapeChar
			dst[di+1] = val
			di += 2
		}
		si += 2
		// find next escape char
		n := strings.IndexByte(src[si:], e.escapeChar)
		if n == -1 {
			n = len(src) - si
		}
		// copy all chars before next escape char
		di += copy(dst[di:], src[si:si+n])
		si += n
	}
	if si < len(src) {
		di += copy(dst[di:], src[si:])
	}
	return di
}
