package run

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/relex/gotils/channels"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/defs"
)

// loaderIface defines abstract configuration loader
//
// It's implemented by Loader and Reloader, and only used by run.Run
//
// TODO: remove interface and call Reloader direcrly after it's proved stable
type loaderIface interface {
	// StartOrchestrator launches an Orchestrator in background and returns it
	StartOrchestrator(ologger logger.Logger) base.Orchestrator

	// LaunchInputs starts all inputs in background and returns (list of addresses, shutdown function)
	LaunchInputs(orchestrator base.Orchestrator) ([]string, func())

	// GetConfigStats returns the ConfigStats from parsing the config file
	GetConfigStats() ConfigStats

	// GetMetricGatherer returns a metric gatherer containing default metrics and metrics from both input and pipeline(s)
	//
	// If called before orchestrator or inputs are launched, the gatherer would not collect metrics from missing parts created later
	GetMetricGatherer() prometheus.Gatherer

	// GetMetricQuerier returns a composite querier that can locate both input metrics and pipeline metrics
	//
	// If called before orchestrator or inputs are launched, the querier would not find metrics from missing parts created later
	GetMetricQuerier() promreg.MetricQuerier
}

// Loader loads configuration from file and prepares the environments to be launched
//
// Loader should take care of everything derived from the config file, but not trigger anything automatically
//
// Orchstrator and inputs are exposed in place of a simple main loop to allow customization, see Run()
type Loader struct {
	Config
	ConfigStats ConfigStats

	filepath     string // config file path
	metricPrefix string
	logger       logger.Logger

	PipelineArgs          bconfig.PipelineArgs // parameters for to run a pipeline, may be modified in-place
	inputMetricFactory    *promreg.MetricFactory
	pipelineMetricFactory *promreg.MetricFactory
}

// NewLoaderFromConfigFile creates a new Loader from the given config file
func NewLoaderFromConfigFile(filepath string, metricPrefix string) (*Loader, error) {
	config, schema, stats, configErr := ParseConfigFile(filepath)
	if configErr != nil {
		return nil, configErr
	}

	return &Loader{
		Config:      config,
		ConfigStats: stats,

		filepath:     filepath,
		metricPrefix: metricPrefix,
		logger: logger.WithFields(logger.Fields{
			defs.LabelComponent: "Loader",
			"metricPrefix":      metricPrefix,
		}),

		PipelineArgs: bconfig.PipelineArgs{
			Schema:              schema,
			Deallocator:         base.NewLogAllocator(schema),
			MetricKeyLocators:   schema.MustCreateFieldLocators(config.MetricKeys), // should have been verified in config parsing
			TransformConfigs:    config.Transformations,
			OutputBufferPairs:   config.OutputBuffersPairs,
			NewConsumerOverride: nil,
			SendAllAtEnd:        false,
		},

		inputMetricFactory:    nil,
		pipelineMetricFactory: nil,
	}, nil
}

// StartOrchestrator launches an Orchestrator in background and returns it
func (loader *Loader) StartOrchestrator(ologger logger.Logger) base.Orchestrator {
	if loader.pipelineMetricFactory != nil {
		loader.logger.Panic("StartOrchestrator can only be invoked once per Loader")
	}
	loader.pipelineMetricFactory = promreg.NewMetricFactory(loader.metricPrefix, nil, nil)

	return loader.Orchestration.Value.StartOrchestrator(ologger, loader.PipelineArgs, loader.pipelineMetricFactory)
}

// RelaunchOrchestrator launches the Orchestrator again for recovery in test runs
func (loader *Loader) RelaunchOrchestrator(ologger logger.Logger) base.Orchestrator {
	if loader.pipelineMetricFactory == nil {
		loader.logger.Panic("RelaunchOrchestrator can only be invoked after calling StartOrchestrator and shutting it down")
	}

	return loader.Orchestration.Value.StartOrchestrator(ologger, loader.PipelineArgs, loader.pipelineMetricFactory)
}

// LaunchInputs starts all inputs in background and returns (list of addresses, shutdown function)
//
// The returned input addresses are final, e.g. assigned random port if it's 0 in config file
//
// The return shutdown function only shuts down the inputs, not the orchestrator
func (loader *Loader) LaunchInputs(orchestrator base.Orchestrator) ([]string, func()) {
	if loader.inputMetricFactory != nil {
		loader.logger.Panic("LaunchInputs can only be invoked once per Loader")
	}
	loader.inputMetricFactory = promreg.NewMetricFactory(loader.metricPrefix, nil, nil)

	stopRequest := channels.NewSignalAwaitable()
	inputStoppedSignals := make([]channels.Awaitable, 0, len(loader.Inputs))
	inputAddresses := make([]string, 0, len(loader.Inputs))

	for index, inputConfig := range loader.Inputs {
		input, ierr := inputConfig.Value.NewInput(logger.Root(), loader.PipelineArgs.Deallocator, loader.PipelineArgs.Schema,
			orchestrator, loader.inputMetricFactory, stopRequest)
		if ierr != nil {
			loader.logger.Fatalf("input[%d]: %s", index, ierr.Error())
		}
		input.Start()

		inputAddresses = append(inputAddresses, input.Address())
		inputStoppedSignals = append(inputStoppedSignals, input.Stopped())
	}

	return inputAddresses, func() {
		stopRequest.Signal()
		channels.AllAwaitables(inputStoppedSignals...).WaitForever()
	}
}

// GetConfigStats returns ConfigStats
func (loader *Loader) GetConfigStats() ConfigStats {
	return loader.ConfigStats
}

// GetMetricGatherer returns a metric gatherer containing default metrics and metrics from both input and pipeline(s)
func (loader *Loader) GetMetricGatherer() prometheus.Gatherer {
	gset := make(prometheus.Gatherers, 0, 3)
	gset = append(gset, prometheus.DefaultGatherer)
	if mf := loader.inputMetricFactory; mf != nil {
		gset = append(gset, mf)
	}
	if mf := loader.pipelineMetricFactory; mf != nil {
		gset = append(gset, mf)
	}
	return gset
}

// GetMetricQuerier returns a composite querier that can locate both input metrics and pipeline metrics
func (loader *Loader) GetMetricQuerier() promreg.MetricQuerier {
	qset := make(promreg.CompositeMetricQuerier, 0, 2)
	if mf := loader.inputMetricFactory; mf != nil {
		qset = append(qset, mf)
	}
	if mf := loader.pipelineMetricFactory; mf != nil {
		qset = append(qset, mf)
	}
	return qset
}
