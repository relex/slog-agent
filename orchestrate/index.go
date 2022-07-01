// Package orchestrate registers the list of all Orchestrator implementations
package orchestrate

import (
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/orchestrate/obykeyset"
	"github.com/relex/slog-agent/orchestrate/osingleton"
)

func init() {
	bconfig.RegisterConfigConstructors(bconfig.OrchestratorConfigCreatorTable{
		"byKeySet":  func() bconfig.OrchestratorConfig { return &obykeyset.Config{} },
		"singleton": func() bconfig.OrchestratorConfig { return &osingleton.Config{} },
	})
}

// Register registers all input config types
func Register() {
	// trigger init()
}
