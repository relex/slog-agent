package run

import (
	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
)

// LoaderIface defines abstract configuration loader
type LoaderIface interface {
	LaunchOrchestrator(ologger logger.Logger) base.Orchestrator
	LaunchInputs(orchestrator base.Orchestrator) ([]string, func())
}

// Loader loads configuration from file and prepares the environments to be launched
//
// Loader should take care of everything derived from the config file, but not trigger anything automatically
//
// Orchstrator and inputs are exposed in place of a simple main loop to allow customization, see Run()
type Loader struct {
	filepath string // config file path

	Config
	MetricFactory *base.MetricFactory
	PipelineArgs  bconfig.PipelineArgs // parameters for to run a pipeline, may be modified in-place
}

func NewLoaderFromConfigFile(filepath string, metricPrefix string) (*Loader, error) {
	config, schema, configErr := ParseConfigFile(filepath)
	if configErr != nil {
		return nil, configErr
	}

	return &Loader{
		filepath: filepath,

		Config:        config,
		MetricFactory: base.NewMetricFactory(metricPrefix, nil, nil),
		PipelineArgs: bconfig.PipelineArgs{
			Schema:              schema,
			Deallocator:         base.NewLogAllocator(schema),
			MetricKeyLocators:   schema.MustCreateFieldLocators(config.MetricKeys), // should have been verified in config parsing
			TransformConfigs:    config.Transformations,
			BufferConfig:        config.Buffer,
			OutputConfig:        config.Output,
			NewConsumerOverride: nil,
			SendAllAtEnd:        false,
		},
	}, nil
}

// LaunchOrchestrator launches an Orchestrator in background and returns it
func (loader *Loader) LaunchOrchestrator(ologger logger.Logger) base.Orchestrator {
	return loader.Orchestration.LaunchOrchestrator(ologger, loader.PipelineArgs, loader.MetricFactory)
}

// LaunchInputs starts all inputs in background and returns (list of addresses, shutdown function)
//
// The returned input addresses are final, e.g. assigned random port if it's 0 in config file
//
// The return shutdown function only shuts down the inputs, not the orchestrator
func (loader *Loader) LaunchInputs(orchestrator base.Orchestrator) ([]string, func()) {
	stopRequest := channels.NewSignalAwaitable()
	inputStoppedSignals := make([]channels.Awaitable, 0, len(loader.Inputs))
	inputAddresses := make([]string, 0, len(loader.Inputs))

	for index, inputConfig := range loader.Inputs {
		input, ierr := inputConfig.NewInput(logger.Root(), loader.PipelineArgs.Deallocator, loader.PipelineArgs.Schema,
			orchestrator, loader.MetricFactory, stopRequest)
		if ierr != nil {
			logger.Fatalf("input[%d]: %s", index, ierr.Error())
		}
		input.Launch()

		inputAddresses = append(inputAddresses, input.Address())
		inputStoppedSignals = append(inputStoppedSignals, input.Stopped())
	}

	return inputAddresses, func() {
		stopRequest.Signal()
		channels.AllAwaitables(inputStoppedSignals...).WaitForever()
	}
}
