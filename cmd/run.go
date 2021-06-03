package cmd

import (
	"context"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/run"
	"github.com/relex/slog-agent/util"
)

type runCommandState struct {
	Config      string `help:"Configuration file path"`
	MetricsAddr string `help:"The listener address to expose Prometheus metrics and debug information"`
	TestMode    bool   `help:"Use test mode config: fast retry and short timeout"`
}

var runCmd runCommandState = runCommandState{
	Config:      "config.yml",
	MetricsAddr: ":9335",
	TestMode:    false,
}

func (cmd *runCommandState) run(args []string) {
	if cmd.TestMode {
		defs.EnableTestMode()
	}

	msrv := util.LaunchMetricsListener(cmd.MetricsAddr)

	run.Run(cmd.Config)

	if err := msrv.Shutdown(context.Background()); err != nil {
		logger.Errorf("error shutting down metrics listener: %v", err)
	}
}
