// Package input registers the list of all LogInput implementations
package input

import (
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/input/sysloginput"
)

func init() {
	bconfig.RegisterLogInputConfigConstructors(map[string]func() bconfig.LogInputConfig{
		"syslog": func() bconfig.LogInputConfig { return &sysloginput.Config{} },
	})
}

// Register registers all input config types
func Register() {
	// trigger init()
}
