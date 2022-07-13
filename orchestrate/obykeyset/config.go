package obykeyset

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/orchestrate/obase"
	"golang.org/x/exp/maps"
)

// Config defines the configuration for ByKeySet Orchestrator
type Config struct {
	bconfig.Header `yaml:",inline"`
	Keys           []string `yaml:"keys"` // Key field names
	TagTemplate    string   `yaml:"tag"`  // Tag is evaluated at the creation of new pipeline and can only reference key fields
}

// StartOrchestrator constructs and launches a by-keySet orchestrator and pipeline(s)
func (cfg *Config) StartOrchestrator(parentLogger logger.Logger, args bconfig.PipelineArgs, metricCreator promreg.MetricCreator) base.Orchestrator {
	startPipeline := obase.PrepareSequentialPipeline(args)

	// to create pipelines for queued logs on disk immediately after starting, otherwise queues can't be processed
	// until clients send logs from the same source to trigger the recreation of their corresponding pipelines
	initialPipelineIDs := make(map[string]struct{})
	for _, pair := range args.OutputBufferPairs {
		ids := pair.BufferConfig.ListBufferIDs(parentLogger, pair.OutputConfig.MatchChunkID, metricCreator.AddOrGetPrefix("recovery_", nil, nil))

		for _, id := range ids {
			initialPipelineIDs[id] = struct{}{}
		}
	}

	return NewOrchestrator(parentLogger, args.Schema, cfg.Keys, cfg.TagTemplate, metricCreator, startPipeline, maps.Keys(initialPipelineIDs))
}

// VerifyConfig verifies orchestration config and returns key fields
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

	return cfg.Keys, nil
}
