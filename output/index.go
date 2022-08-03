// Package output registers the list of all output implementations
package output

import (
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/output/datadog"
	"github.com/relex/slog-agent/output/fluentdforward"
)

func init() {
	bconfig.RegisterConfigConstructors(bconfig.LogOutputConfigCreatorTable{
		"datadog":        func() bconfig.LogOutputConfig { return &datadog.Config{} },
		"fluentdForward": func() bconfig.LogOutputConfig { return &fluentdforward.Config{} },
	})
}

// Register registers all input config types
func Register() {
	// trigger init()
}
