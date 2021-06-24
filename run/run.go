// Package run runs the actual log agent
package run

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/defs"
)

// Run runs the agent until stopped by signals
func Run(configFile string) {
	var loader LoaderIface
	var loaderErr error
	if defs.EnableConfigReload {
		loader, loaderErr = NewReloaderFromConfigFile(configFile, "slogagent_")
	} else {
		loader, loaderErr = NewLoaderFromConfigFile(configFile, "slogagent_")
	}
	if loaderErr != nil {
		logger.Fatal(loaderErr)
	}

	orchestrator := loader.LaunchOrchestrator(logger.Root())
	_, shutdownInputs := loader.LaunchInputs(orchestrator)

	runLogger := logger.WithField(defs.LabelComponent, "Launcher")

	// wait for shutdown signal
	{
		sigChan := make(chan os.Signal, 10)
		signal.Notify(sigChan, syscall.SIGINT)
		signal.Notify(sigChan, syscall.SIGTERM)
		s := <-sigChan
		runLogger.Infof("received %s, shutting down", s)
	}

	shutdownInputs()
	orchestrator.Shutdown()
	runLogger.Info("clean exit")
}
