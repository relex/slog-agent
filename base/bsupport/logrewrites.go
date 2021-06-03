package bsupport

import (
	"fmt"

	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// NewRewritersFromConfig creates rewriters from a list of rewriter configurations and returns the head
func NewRewritersFromConfig(rewriterConfigs []bconfig.LogRewriterConfigHolder, schema base.LogSchema) base.LogRewriter {
	if len(rewriterConfigs) == 0 {
		return nil
	}
	head := base.LogRewriter(nil)
	for i := len(rewriterConfigs) - 1; i >= 0; i-- {
		head = rewriterConfigs[i].NewRewriter(schema, head)
	}
	return head
}

// VerifyRewriterConfigs verifies a list of rewriter configurations
func VerifyRewriterConfigs(rewriterConfigs []bconfig.LogRewriterConfigHolder, schema base.LogSchema, header string) error {
	lastI := len(rewriterConfigs) - 1
	for i, rwc := range rewriterConfigs {
		err := rwc.VerifyConfig(schema, i < lastI)
		if err != nil {
			return fmt.Errorf("%s[%d] %s: %w", header, i, rwc.Location, err)
		}
	}
	return nil
}
