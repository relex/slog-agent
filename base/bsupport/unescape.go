package bsupport

import (
	"github.com/relex/slog-agent/util/stringunescape"
)

// NewSyslogUnescaper creates an unescaper for syslog inputs
// Unescaping is logically part of input parsing but moved to transform and rewrite for performance reason
func NewSyslogUnescaper() stringunescape.Unescaper {
	return stringunescape.NewUnescaper('\\', map[byte]byte{
		'b': '\b',
		'f': '\f',
		'n': '\n',
		'r': '\r',
		't': '\t',
	})
}
