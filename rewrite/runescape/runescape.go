// Package runescape provides 'unescape' rewriter, which handles custom escape characters like those in JSON strings.
//
// The rewriter equals to 'unescape' transform, except it's done in serialization stage with no extra string copy
// needed, which is critical for large messages (e.g. error dumps).
//
// Note using unescape rewriter instead of unescape transform can mess up transforms which depend on the results of
// unescaping.
//
// The 'unescape' rewriter serves as the last rewriter in the chain, in place of 'copy'
//nolint:revive
package runescape

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
)

// Config for unescapeRewriter
type Config struct {
	bconfig.Header `yaml:",inline"`
}

type unescapeRewriter struct{}

var unescaper = bsupport.NewSyslogUnescaper()

// NewRewriter creates unescapeRewriter
func (c *Config) NewRewriter(schema base.LogSchema, next base.LogRewriter) base.LogRewriter {
	if next != nil {
		logger.Panic("'unescape' must be the last rewriter")
	}
	return &unescapeRewriter{}
}

// VerifyConfig verifies unescapeRewriter config
func (c *Config) VerifyConfig(schema base.LogSchema, hasNext bool) error {
	if hasNext {
		return fmt.Errorf("'unescape' must be the last rewriter")
	}
	return nil
}

func (rw *unescapeRewriter) MaxFieldLength(value string, record *base.LogRecord) int {
	return len(value)
}

func (rw *unescapeRewriter) WriteFieldBody(value string, record *base.LogRecord, buffer []byte) int {
	if record.Unescaped {
		return copy(buffer, value)
	}
	record.Unescaped = true
	first := unescaper.FindFirst(value)
	if first == -1 {
		return copy(buffer, value)
	}
	return unescaper.RunToBuffer(value, first, buffer)
}
