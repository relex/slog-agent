package osingleton

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promreg"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/orchestrate/obase"
)

// Config defines the configuration for Singleton Orchestrator
type Config struct {
	bconfig.Header `yaml:",inline"`
	Tag            string `yaml:"tag"`
}

// StartOrchestrator constructs and launches a singleton orchestrator and the pipeline
func (cfg *Config) StartOrchestrator(parentLogger logger.Logger, args bconfig.PipelineArgs, metricCreator promreg.MetricCreator) base.Orchestrator {
	launchPipeline := obase.PrepareSequentialPipeline(args)
	pipelineMetricCreator := metricCreator.AddOrGetPrefix("process_", []string{"orchestrator"}, []string{"singleton"})
	return NewOrchestrator(parentLogger, cfg.Tag, pipelineMetricCreator, launchPipeline)
}

// VerifyConfig verifies orchestration config
func (cfg *Config) VerifyConfig(schema base.LogSchema) ([]string, error) {
	if len(cfg.Tag) == 0 {
		return nil, fmt.Errorf(".tag is unspecified")
	}
	// no variable accepted atm
	if _, terr := obase.NewTagBuilder(cfg.Tag, nil); terr != nil {
		return nil, fmt.Errorf(".tag: %w", terr)
	}
	return nil, nil
}
