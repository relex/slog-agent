package run

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
	"golang.org/x/exp/slices"
)

// Reloader overrides Loader to support configuration reloading
//
// It provides the underlying configuration reloading but not responsible to trigger or to use the reloaded setup
type Reloader struct {
	*Loader

	reloadingLock *sync.Mutex
	numReload     int64
}

// NewReloaderFromConfigFile creates a new Reloader with a new Loader from the given config file
func NewReloaderFromConfigFile(filepath string, metricPrefix string) (*Reloader, error) {
	// launch the orchestrator which manages pipelines
	loader, confErr := NewLoaderFromConfigFile(filepath, metricPrefix)
	if confErr != nil {
		return nil, confErr // return error if config file is invalid at startup (NOT when reloading)
	}

	return &Reloader{
		Loader: loader,

		reloadingLock: &sync.Mutex{},
		numReload:     0,
	}, nil
}

// StartOrchestrator launches a reloadable Orchestrator
func (reloader *Reloader) StartOrchestrator(ologger logger.Logger) base.Orchestrator {
	firstDownstreamOrchestrator := reloader.Loader.StartOrchestrator(ologger)

	return NewReloadableOrchestrator(firstDownstreamOrchestrator, reloader.initiateDownstreamReload)
}

func (reloader *Reloader) initiateDownstreamReload() (CompleteReloadingFunc, error) {
	newLoader, confErr := NewLoaderFromConfigFile(reloader.filepath, reloader.metricPrefix)
	if confErr != nil {
		return nil, confErr
	}
	if err := checkConfigCompatibility(
		reloader.Loader.Config, reloader.PipelineArgs.Schema, reloader.Loader.ConfigStats,
		newLoader.Config, newLoader.PipelineArgs.Schema, newLoader.ConfigStats); err != nil {
		return nil, err
	}
	newLoader.ConfigStats.Log(reloader.logger)

	return func() base.Orchestrator {
		reloader.reloadingLock.Lock()
		defer reloader.reloadingLock.Unlock()

		newLoader.PipelineArgs.Deallocator = reloader.Loader.PipelineArgs.Deallocator
		newLoader.inputMetricFactory = reloader.Loader.inputMetricFactory // which also forbids new inputs from being launched
		newLoader.pipelineMetricFactory = nil                             // will be new

		reloader.Loader = newLoader
		reloader.numReload++
		return reloader.Loader.StartOrchestrator(reloader.logger.WithField("numReload", reloader.numReload))
	}, nil
}

// GetMetricGatherer returns a metric gatherer containing default metrics and metrics from both input and pipeline(s)
//
// Unlike Loader's GetMetricGatherer which returns a composite Gatherer containing current metric factories, Reloader's
// returns a GathererFunc which always gathers from the latest metric factories, if they've been recreated by reloading
func (reloader *Reloader) GetMetricGatherer() prometheus.Gatherer {
	return prometheus.GathererFunc(func() ([]*dto.MetricFamily, error) {
		return reloader.Loader.GetMetricGatherer().Gather()
	})
}

func checkConfigCompatibility(
	oldConf Config, oldSchema base.LogSchema, oldStats ConfigStats,
	newConf Config, newSchema base.LogSchema, newStats ConfigStats,
) error {
	{
		oldMax := oldConf.Schema.MaxFields
		newMax := newConf.Schema.MaxFields
		if oldMax != newMax {
			return fmt.Errorf("schema/maxFields must not change: old=%d, new=%d", oldMax, newMax)
		}
	}
	{
		var err error
		var oldInputs, newInputs string
		if oldInputs, err = util.MarshalYaml(oldConf.Inputs); err != nil {
			return fmt.Errorf("inputs: failed to marshal old config: %w", err)
		}
		if newInputs, err = util.MarshalYaml(newConf.Inputs); err != nil {
			return fmt.Errorf("inputs: failed to marshal new config: %w", err)
		}
		if oldInputs != newInputs {
			return fmt.Errorf("inputs must not change: old=%s", oldInputs)
		}
	}
	{
		oldType := oldConf.Orchestration.Value.GetType()
		newType := newConf.Orchestration.Value.GetType()
		if oldType != newType {
			return fmt.Errorf("orchestration/type must not change: old=%s, new=%s", oldType, newType)
		}
	}
	{
		oldKeys := oldStats.OrchestrationKeys
		newKeys := newStats.OrchestrationKeys
		if !slices.Equal(oldKeys, newKeys) {
			return fmt.Errorf("orchestration/keys must not change: old=%s, new=%s", oldKeys, newKeys)
		}
	}

	// check schema fields last because other comparisons are more verbose
	{
		for _, field := range oldStats.FixedFields {
			oldLocation := oldSchema.MustCreateFieldLocator(field)
			newLocation, newErr := newSchema.CreateFieldLocator(field)
			if newErr != nil {
				return fmt.Errorf("schema/fields: required field \"%s\" is missing from the new schema; position=%dth",
					field, oldLocation+1)
			}
			if oldLocation != newLocation {
				return fmt.Errorf("schema/fields: required field \"%s\" has been moved, from=%dth to=%dth",
					field, oldLocation+1, newLocation+1)
			}
		}
	}
	return nil
}
