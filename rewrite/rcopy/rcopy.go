// Package rcopy provides 'copy' rewriter, which copies the original field value unmodified
//
// The 'copy' rewriter serves as the last rewriter in the chain, if no extra processing is needed
//nolint:revive
package rcopy

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// Config for copyRewriter
type Config struct {
	bconfig.Header `yaml:",inline"`
}

type copyRewriter struct{}

// NewRewriter creates copyRewriter for test
func NewRewriter() base.LogRewriter {
	return &copyRewriter{}
}

// NewRewriter creates copyRewriter
func (c *Config) NewRewriter(schema base.LogSchema, next base.LogRewriter) base.LogRewriter {
	if next != nil {
		logger.Panic("'copy' must be the last rewriter")
	}
	return &copyRewriter{}
}

// VerifyConfig verifies copyRewriter config
func (c *Config) VerifyConfig(schema base.LogSchema, hasNext bool) error {
	if hasNext {
		return fmt.Errorf("'copy' must be the last rewriter")
	}
	return nil
}

func (rw *copyRewriter) MaxFieldLength(value string, record *base.LogRecord) int {
	return len(value)
}

func (rw *copyRewriter) WriteFieldBody(value string, record *base.LogRecord, buffer []byte) int {
	return copy(buffer, value)
}
