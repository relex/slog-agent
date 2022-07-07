// Package run runs the actual log agent
package run

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/defs"
)

// Run runs the agent until stopped by signals
func Run(configFile string, metricAddress string, allowReload bool) {
	rlogger := logger.WithField(defs.LabelComponent, "Run")

	var loader loaderIface
	var confErr error
	if allowReload {
		loader, confErr = NewReloaderFromConfigFile(configFile, "slogagent_")
	} else {
		loader, confErr = NewLoaderFromConfigFile(configFile, "slogagent_")
	}
	if confErr != nil {
		rlogger.Fatal(confErr)
	}
	loader.GetConfigStats().Log(rlogger)

	orchestrator := loader.StartOrchestrator(logger.Root())
	_, shutdownInputs := loader.LaunchInputs(orchestrator)

	msrv := promreg.LaunchMetricListener(metricAddress, loader.GetMetricGatherer(), true)

	// wait for shutdown signal
	{
		sigChan := make(chan os.Signal, 10)
		signal.Notify(sigChan, syscall.SIGINT)
		signal.Notify(sigChan, syscall.SIGTERM)
		s := <-sigChan
		rlogger.Infof("received %s, shutting down", s)
	}

	shutdownInputs()
	orchestrator.Shutdown()
	rlogger.Info("clean exit")

	if err := msrv.Shutdown(context.Background()); err != nil {
		rlogger.Errorf("failed to shut down metrics listener: %v", err)
	}
}
