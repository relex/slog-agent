package obykeyset

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/orchestrate/obase"
)

// Config defines the configuration for ByKeySet Orchestrator
type Config struct {
	bconfig.Header `yaml:",inline"`
	Keys           []string `yaml:"keys"` // Key field names
	TagTemplate    string   `yaml:"tag"`  // Tag is evaluated at the creation of new pipeline and can only reference key fields
	NumWorkers     int      `yaml:"num"`  // Numbers of workers. N > 1 creates N pipelines per key-set for parallel processing
}

// LaunchOrchestrator constructs and launches a by-keySet orchestrator and pipeline(s)
func (cfg *Config) LaunchOrchestrator(parentLogger logger.Logger, args bconfig.PipelineArgs, metricFactory *base.MetricFactory) base.Orchestrator {
	var launchPipeline base.PipelineWorkersLauncher
	if cfg.NumWorkers == 1 {
		launchPipeline = bsupport.NewSequentialPipelineLauncher(args)
	} else {
		launchDistributor := func(parentLogger logger.Logger, input <-chan []*base.LogRecord, tag string,
			subMetricFactory *base.MetricFactory, launchChildWorkers base.OrderedPipelineWorkersLauncher, onStopped func()) {
			distributor := obase.NewDistributor(parentLogger, input, tag, cfg.NumWorkers, subMetricFactory, launchChildWorkers)
			distributor.Launch()
			distributor.Stopped().Next(onStopped)
		}
		launchPipeline = bsupport.NewParallelPipelineLauncher(args, launchDistributor)
	}

	existingPipelineIDs := args.BufferConfig.ListBufferIDs(parentLogger, args.OutputConfig.MatchChunkID,
		metricFactory.NewSubFactory("recovery_", nil, nil))

	return NewOrchestrator(parentLogger, args.Schema, cfg.Keys, cfg.TagTemplate, metricFactory, launchPipeline, existingPipelineIDs)
}

// VerifyConfig verifies orchestration config
func (cfg *Config) VerifyConfig(schema base.LogSchema) ([]string, error) {
	if len(cfg.Keys) == 0 {
		return nil, fmt.Errorf(".keys is empty")
	}
	if _, lerr := schema.CreateFieldLocators(cfg.Keys); lerr != nil {
		return nil, fmt.Errorf(".keys: %w", lerr)
	}

	if len(cfg.TagTemplate) == 0 {
		return nil, fmt.Errorf(".tag is unspecified")
	}
	if _, terr := obase.NewTagBuilder(cfg.TagTemplate, cfg.Keys); terr != nil {
		return nil, fmt.Errorf(".tag: %w", terr)
	}

	if cfg.NumWorkers == 0 {
		return nil, fmt.Errorf(".num is unspecified")
	}
	if cfg.NumWorkers <= 0 {
		return nil, fmt.Errorf(".num must be at least 1: %d", cfg.NumWorkers)
	}

	metricKeyNames := make([]string, len(cfg.Keys))
	for i, key := range cfg.Keys {
		metricKeyNames[i] = "key_" + key
	}
	return metricKeyNames, nil
}
