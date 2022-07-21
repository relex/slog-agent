package cmd

import (
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/run"
)

type runCommandState struct {
	Config      string `help:"Configuration file path"`
	MetricsAddr string `help:"The HTTP listener address to expose Prometheus metrics and debug information. Use :0 for auto assignment."`
	AllowReload bool   `help:"Allow configuration reloading by SIGHUP"`
	TestMode    bool   `help:"Use test mode config: fast retry and short timeout"`
}

var runCmd = runCommandState{
	Config:      "config.yml",
	MetricsAddr: ":9335",
	AllowReload: false,
	TestMode:    false,
}

//nolint:revive
func (cmd *runCommandState) run(args []string) {
	if cmd.TestMode {
		defs.EnableTestMode()
	}

	run.Run(cmd.Config, cmd.MetricsAddr, cmd.AllowReload)
}
