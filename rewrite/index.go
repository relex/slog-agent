// Package rewrite registers the list of all LogRewriter implementations
package rewrite

import (
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/rewrite/rcopy"
	"github.com/relex/slog-agent/rewrite/rinline"
	"github.com/relex/slog-agent/rewrite/runescape"
)

func init() {
	bconfig.RegisterConfigConstructors(bconfig.LogRewriterConfigCreatorTable{
		"copy":     func() bconfig.LogRewriterConfig { return &rcopy.Config{} },
		"inline":   func() bconfig.LogRewriterConfig { return &rinline.Config{} },
		"unescape": func() bconfig.LogRewriterConfig { return &runescape.Config{} },
	})
}

// Register registers all rewriter config types
func Register() {
	// trigger init()
}
