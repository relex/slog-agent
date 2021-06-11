// Package run runs the actual log agent
package run

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/defs"
)

// Run runs the agent until stopped by signals
func Run(configFile string) {
	runLogger := logger.WithField(defs.LabelComponent, "Launcher")
	agentLogger := logger.Root()

	config, schema, cerr := LoadConfigFile(configFile)
	if cerr != nil {
		runLogger.Fatalf("config: %s", cerr.Error())
	}

	allocator := base.NewLogAllocator(schema)
	pipelineArgs := bconfig.PipelineArgs{
		Schema:              schema,
		Deallocator:         allocator,
		MetricKeyLocators:   schema.MustCreateFieldLocators(config.MetricKeys),
		TransformConfigs:    config.Transformations,
		BufferConfig:        config.Buffer,
		OutputConfig:        config.Output,
		NewConsumerOverride: nil,
		SendAllAtEnd:        false,
	}
	metricFactory := base.NewMetricFactory("slogagent_", nil, nil)

	// launch the orchestrator which manages pipelines
	orchestrator := config.Orchestration.LaunchOrchestrator(agentLogger, pipelineArgs, metricFactory)

	// launch inputs
	inputShutdownSignals := make([]channels.Awaitable, 0, len(config.Inputs))
	stopRequest := channels.NewSignalAwaitable()

	for index, inputConfig := range config.Inputs {
		input, ierr := inputConfig.NewInput(agentLogger, allocator, schema, orchestrator, metricFactory, stopRequest)
		if ierr != nil {
			runLogger.Fatalf("input[%d]: %s", index, ierr.Error())
		}
		input.Launch()
		inputShutdownSignals = append(inputShutdownSignals, input.Stopped())
	}

	// wait for shutdown signal
	{
		sigChan := make(chan os.Signal, 10)
		signal.Notify(sigChan, syscall.SIGINT)
		signal.Notify(sigChan, syscall.SIGTERM)
		s := <-sigChan
		runLogger.Infof("received %s, shutting down", s)
	}

	// shutdown
	stopRequest.Signal()
	channels.AllAwaitables(inputShutdownSignals...).WaitForever()
	runLogger.Info("clean exit")
}
